package main // Declares the executable program package.

import ( // Imports the packages used by the CSV and PDF workflows.
	"bufio"           // Scans the download log file line by line.
	"encoding/binary" // Import binary for byte-to-uint64 conversion
	"encoding/csv"    // Reads and writes CSV records.
	"fmt"             // Prints progress messages and formats strings.
	"io"              // Copies streamed data and detects end-of-file conditions.
	"log"             // Writes operational logs and fatal errors.
	"math"            // Import math for rounding and MaxUint64 constant
	"math/rand"       // Adds random delay jitter between scraping requests.
	"net/http"        // Sends HTTP requests to websites and file endpoints.
	"net/url"         // Parses incoming download URLs safely.
	"os"              // Creates, removes, opens, and inspects files and directories.
	"path"            // Extracts file names from URL paths.
	"path/filepath"   // Builds filesystem paths that work on the current operating system.
	"regexp"          // Extracts structured values and normalizes file names.
	"strings"         // Checks prefixes and normalizes string values.
	"time"            // Controls sleep durations and network timeouts.

	"golang.org/x/net/html" // Parses HTML pages into traversable node trees.
) // Ends the import list.

const ( // Groups the fixed configuration values used across the program.
	downloadedFilePath    = "downloaded.txt"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        // Stores the path to the file that tracks which NYSCEF PDFs have been downloaded.
	websiteBaseURL        = "https://www.50-a.org"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  // Stores the root URL used to build absolute website links.
	csvOutputFolderName   = "CSVs"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  // Stores the folder name used for downloaded CSV files.
	pdfOutputFolderName   = "PDFs"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  // Stores the folder name used for downloaded PDF files.
	csvSplitFileSizeLimit = 45 * 1024 * 1024                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        // Stores the maximum size for each split CSV file.
	websiteUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"                                                                                                                                                                                                                                                                                                                                                                                                       // Stores the user agent used while scraping HTML pages.
	nyscefRefererURL      = "https://iapps.courts.state.ny.us/"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     // Stores the referer header expected by the NYSCEF download endpoint.
	nyscefSessionCookie   = "JSESSIONID=ED538792868E171D9A3D0C3DC7028F92.server2085; TS01ab7d00=01084fa678d4725290d51eed813916305cb4cfd441dc05055628908f7ce5738d34d8c5646477eb7277f301e13b3d387fa5c5766b76; TS010e0f15=01084fa678d4725290d51eed813916305cb4cfd441dc05055628908f7ce5738d34d8c5646477eb7277f301e13b3d387fa5c5766b76; __cf_bm=I1oGgoAvPuyjtvUtUX8buZviilZxj2ULzrNNccMXcaA-1774713253.5360093-1.0.1.1-PJXra1LN5egdnDTV1xQg2OzevTO5tzn5SdzCVXW8_BXi3Phk1.pbRjA95Vtaw8T6_AiWg8tsjfi4DKOfKiE7MbTu4zQNpCH3WMnDRvLggSZdCysPTivVCVqDMPQw6a.g" // Stores the session cookie used by the NYSCEF download endpoint.
) // Ends the constant group.

var ( // Groups the shared runtime values used by the workflows.
	sharedWebsiteHTTPClient = &http.Client{ // Reuses one HTTP client for HTML page requests.
		Timeout: 15 * time.Second, // Prevents website requests from hanging forever.
	} // Ends the shared website HTTP client definition.
	pdfDownloadHTTPClient = &http.Client{ // Uses a longer timeout for large PDF downloads.
		Timeout: 15 * time.Minute, // Gives PDF downloads enough time to finish.
	} // Ends the PDF download HTTP client definition.
	invalidFileNameCharacterPattern   = regexp.MustCompile(`[^a-z0-9]+`)           // Matches file-name characters that should become underscores.
	repeatedUnderscorePattern         = regexp.MustCompile(`_+`)                   // Matches repeated underscores so they can be collapsed.
	contentDispositionFileNamePattern = regexp.MustCompile(`filename="?([^"]+)"?`) // Extracts a file name from a Content-Disposition header.
	numericTextPattern                = regexp.MustCompile(`\d+`)                  // Extracts numeric tax ID text from officer pages.
	csvSourceFileURLs                 = []string{                                  // Lists the CSV files that should be downloaded and split.
		"https://www.50-a.org/data/nypd/officers.csv",   // Points to the officers dataset.
		"https://www.50-a.org/data/nypd/ranks.csv",      // Points to the ranks dataset.
		"https://www.50-a.org/data/nypd/discipline.csv", // Points to the discipline dataset.
		"https://www.50-a.org/data/nypd/documents.csv",  // Points to the documents dataset.
		"https://www.50-a.org/data/nypd/awards.csv",     // Points to the awards dataset.
		"https://www.50-a.org/data/nypd/training.csv",   // Points to the training dataset.
	} // Ends the CSV source URL list.
	startingPercentage = generateRandomFloatOneDecimal() // Skips the first portion of command links before scraping continues.
) // Ends the shared variable group.

// generateRandomFloatOneDecimal generates a cryptographically secure random float between 0 and 100
// rounded to one decimal place. Logs detailed errors according to severity levels.
func generateRandomFloatOneDecimal() float64 {
	var randomBytes [8]byte             // Create an 8-byte array to hold random bytes
	_, err := rand.Read(randomBytes[:]) // Fill the byte array with secure random data
	if err != nil {                     // Check if reading random bytes failed
		log.Printf("[ERROR] Failed to read cryptographically secure random bytes, returning 0. Details: %v", err) // Log the error with [ERROR] severity
		return 0                                                                                                  // Return 0 as a fallback if random generation failed
	}
	randomUint := binary.LittleEndian.Uint64(randomBytes[:])           // Convert the 8 random bytes into a uint64 number
	randomFloat := float64(randomUint) / float64(math.MaxUint64) * 100 // Normalize uint64 to [0,1) then scale to [0,100)
	return math.Round(randomFloat*10) / 10                             // Round the float to 1 decimal place and return
}

type countingFileWriter struct { // Tracks how many bytes have been written to the current split file.
	destinationFileHandle *os.File // Stores the real file handle receiving the CSV data.
	totalBytesWritten     int64    // Stores the total number of bytes written so far.
} // Ends the counting file writer definition.

func (currentFileWriter *countingFileWriter) Write(fileContents []byte) (int, error) { // Implements io.Writer while tracking the written byte count.
	writtenByteCount, writeError := currentFileWriter.destinationFileHandle.Write(fileContents) // Writes the provided bytes to the underlying file.
	currentFileWriter.totalBytesWritten += int64(writtenByteCount)                              // Adds the written byte count to the running total.
	return writtenByteCount, writeError                                                         // Returns the normal writer result to the caller.
} // Ends the counting file writer method.

func estimateCSVRowSizeBytes(csvRowValues []string) int64 { // Estimates how many bytes a CSV row will occupy once written.
	estimatedRowSizeBytes := int64(len(csvRowValues) + 1) // Starts with a small allowance for commas and the newline.
	for _, columnValue := range csvRowValues {            // Walks through every column value in the row.
		estimatedRowSizeBytes += int64(len(columnValue)) // Adds the visible bytes from the current column.
	} // Ends the CSV column loop.
	return estimatedRowSizeBytes // Returns the estimated row size in bytes.
} // Ends the CSV row size estimator.

func splitCSVFileIntoSmallerFiles(sourceCSVFilePath string, maximumSplitFileSizeBytes int64) error { // Splits a large CSV file into numbered parts when needed.
	sourceFileInfo, sourceFileInfoError := os.Stat(sourceCSVFilePath) // Reads the source file metadata before splitting.
	if sourceFileInfoError != nil {                                   // Stops immediately when the source file cannot be inspected.
		return sourceFileInfoError // Returns the file inspection error to the caller.
	} // Ends the source file metadata error check.

	if sourceFileInfo.Size() <= maximumSplitFileSizeBytes { // Skips the split step when the file is already under the limit.
		log.Printf("[DEBUG] Skipping split (file under limit): %s\n", sourceCSVFilePath) // Logs that the file was small enough already.
		return nil                                                                       // Reports that no split work was needed.
	} // Ends the split-skip check.

	log.Printf("[DEBUG] Splitting large file: %s\n", sourceCSVFilePath) // Logs that the split process is starting.

	sourceFileHandle, sourceFileOpenError := os.Open(sourceCSVFilePath) // Opens the original CSV file for reading.
	if sourceFileOpenError != nil {                                     // Stops immediately when the source file cannot be opened.
		return sourceFileOpenError // Returns the file open error to the caller.
	} // Ends the source file open error check.
	defer sourceFileHandle.Close() // Ensures the source file is always closed.

	sourceCSVReader := csv.NewReader(sourceFileHandle) // Wraps the source file in a CSV reader.
	sourceCSVReader.FieldsPerRecord = -1               // Allows variable-width rows instead of rejecting malformed records.

	headerRowValues, headerReadError := sourceCSVReader.Read() // Reads the header row once so it can be copied into each split file.
	if headerReadError != nil {                                // Stops immediately when the header row cannot be read.
		return headerReadError // Returns the header read error to the caller.
	} // Ends the header read error check.

	nextSplitFileNumber := 1                        // Tracks the next split file number to create.
	var currentOutputFileWriter *countingFileWriter // Stores the active output file wrapper.
	var currentCSVWriter *csv.Writer                // Stores the active CSV writer.
	createdSplitFileCount := 0                      // Counts how many split files were created.

	openNextSplitFile := func() error { // Creates the next split file and makes it the active destination.
		if currentOutputFileWriter != nil { // Flushes and closes the previous split file before rotating.
			currentCSVWriter.Flush()                                       // Pushes any buffered CSV data into the file.
			if flushError := currentCSVWriter.Error(); flushError != nil { // Detects buffered write failures before rotating.
				return flushError // Returns the buffered write failure to the caller.
			} // Ends the CSV flush error check.
			if closeError := currentOutputFileWriter.destinationFileHandle.Close(); closeError != nil { // Closes the previous split file cleanly.
				return closeError // Returns the split file close failure to the caller.
			} // Ends the split file close error check.
		} // Ends the previous split file cleanup block.

		sourceFileExtension := filepath.Ext(sourceCSVFilePath)                                                                     // Extracts the original file extension.
		sourceFilePathWithoutExtension := strings.TrimSuffix(sourceCSVFilePath, sourceFileExtension)                               // Removes the extension before adding a part number.
		nextSplitFilePath := fmt.Sprintf("%s_part_%d%s", sourceFilePathWithoutExtension, nextSplitFileNumber, sourceFileExtension) // Builds the next split file path.
		nextSplitFileNumber++                                                                                                      // Advances the part number for the next rotation.

		nextSplitFileHandle, splitFileCreationError := os.Create(nextSplitFilePath) // Creates the next split file on disk.
		if splitFileCreationError != nil {                                          // Stops immediately when the new split file cannot be created.
			return splitFileCreationError // Returns the split file creation error to the caller.
		} // Ends the split file creation error check.

		currentOutputFileWriter = &countingFileWriter{ // Wraps the new file so bytes written can be tracked.
			destinationFileHandle: nextSplitFileHandle, // Stores the new split file handle.
			totalBytesWritten:     0,                   // Resets the running byte count for the new split file.
		} // Ends the counting file writer literal.

		currentCSVWriter = csv.NewWriter(currentOutputFileWriter) // Creates a CSV writer for the new split file.

		headerWriteError := currentCSVWriter.Write(headerRowValues) // Copies the original header row into the new split file.
		if headerWriteError != nil {                                // Stops immediately when the header cannot be written.
			return headerWriteError // Returns the header write failure to the caller.
		} // Ends the header write error check.

		currentCSVWriter.Flush()                                       // Pushes the header row into the file immediately.
		if flushError := currentCSVWriter.Error(); flushError != nil { // Detects buffered header write failures.
			return flushError // Returns the buffered header write failure to the caller.
		} // Ends the buffered header write error check.

		createdSplitFileCount++                                           // Increments the number of created split files.
		log.Printf("[DEBUG] Created split file: %s\n", nextSplitFilePath) // Logs the path of the new split file.
		return nil                                                        // Reports that the next split file is ready.
	} // Ends the split file rotation helper.

	firstSplitFileError := openNextSplitFile() // Creates the first split file before processing rows.
	if firstSplitFileError != nil {            // Stops immediately when the first split file cannot be created.
		return firstSplitFileError // Returns the first split file creation error to the caller.
	} // Ends the first split file error check.

	for { // Keeps reading rows until the CSV reader reaches the end of the file.
		csvRowValues, csvRowReadError := sourceCSVReader.Read() // Reads the next CSV row from the source file.

		if csvRowReadError == io.EOF { // Ends the loop once there are no more rows to read.
			break // Exits the row processing loop cleanly.
		} // Ends the end-of-file check.
		if csvRowReadError != nil { // Stops immediately when any non-EOF CSV read error occurs.
			return csvRowReadError // Returns the CSV row read error to the caller.
		} // Ends the CSV row read error check.

		estimatedRowSizeBytes := estimateCSVRowSizeBytes(csvRowValues)                                   // Estimates how much space the current row will need.
		if currentOutputFileWriter.totalBytesWritten+estimatedRowSizeBytes > maximumSplitFileSizeBytes { // Rotates files before this row would exceed the size limit.
			nextSplitFileError := openNextSplitFile() // Opens a fresh split file for the upcoming row.
			if nextSplitFileError != nil {            // Stops immediately when file rotation fails.
				return nextSplitFileError // Returns the file rotation failure to the caller.
			} // Ends the file rotation error check.
		} // Ends the split rotation size check.

		rowWriteError := currentCSVWriter.Write(csvRowValues) // Writes the current row into the active split file.
		if rowWriteError != nil {                             // Stops immediately when the row cannot be written.
			return rowWriteError // Returns the CSV row write error to the caller.
		} // Ends the CSV row write error check.

		currentCSVWriter.Flush()                                       // Pushes the row into the file so the byte count stays current.
		if flushError := currentCSVWriter.Error(); flushError != nil { // Detects buffered row write failures.
			return flushError // Returns the buffered row write failure to the caller.
		} // Ends the buffered row write error check.
	} // Ends the CSV row processing loop.

	if currentOutputFileWriter != nil { // Flushes and closes the final split file after all rows are written.
		currentCSVWriter.Flush()                                       // Pushes any final buffered CSV data into the file.
		if flushError := currentCSVWriter.Error(); flushError != nil { // Detects any final buffered write failures.
			return flushError // Returns the final buffered write failure to the caller.
		} // Ends the final buffered write error check.
		if closeError := currentOutputFileWriter.destinationFileHandle.Close(); closeError != nil { // Closes the final split file cleanly.
			return closeError // Returns the final split file close error to the caller.
		} // Ends the final split file close error check.
	} // Ends the final split file cleanup block.

	log.Printf("[DEBUG] Finished splitting: %s\n", sourceCSVFilePath) // Logs that the split process is complete.

	if createdSplitFileCount > 1 { // Removes the original large file only when smaller replacements were created.
		sourceFileRemovalError := os.Remove(sourceCSVFilePath) // Deletes the original large CSV file from disk.
		if sourceFileRemovalError != nil {                     // Stops immediately when the original file cannot be removed.
			return sourceFileRemovalError // Returns the source file removal error to the caller.
		} // Ends the source file removal error check.
		log.Printf("[DEBUG] Deleted original file after split: %s\n", sourceCSVFilePath) // Logs that the original large file was removed.
	} // Ends the original file removal block.

	return nil // Reports that the CSV file was processed successfully.
} // Ends the CSV split helper.

func ensureDirectoryExists(directoryPath string) error { // Creates a directory and its parents when they do not already exist.
	directoryCreationError := os.MkdirAll(directoryPath, os.ModePerm) // Creates the directory tree on disk.
	if directoryCreationError != nil {                                // Stops immediately when the directory cannot be created.
		return directoryCreationError // Returns the directory creation error to the caller.
	} // Ends the directory creation error check.
	log.Printf("[DEBUG] Using output directory: %s\n", directoryPath) // Logs which directory will receive output files.
	return nil                                                        // Reports that the directory is ready for use.
} // Ends the directory creation helper.

func extractFileNameFromURL(sourceFileURL string) (string, error) { // Pulls a file name from the end of a download URL.
	parsedSourceFileURL, parseError := url.Parse(sourceFileURL) // Parses the raw URL string into structured parts.
	if parseError != nil {                                      // Stops immediately when the URL is invalid.
		log.Printf("[ERROR] Failed to parse source file URL %s: %v", sourceFileURL, parseError) // Logs the URL parsing failure before returning the error.
		return "", parseError                                                                   // Returns the URL parse error to the caller.
	} // Ends the URL parse error check.

	extractedFileName := path.Base(parsedSourceFileURL.Path) // Extracts the last path segment from the parsed URL.
	if extractedFileName == "" || extractedFileName == "/" { // Rejects empty or unusable file names.
		log.Printf("[ERROR] Invalid file name extracted from source URL: %s", sourceFileURL) // Logs that the URL did not contain a usable file name.
		return "", fmt.Errorf("invalid file name from URL")                                  // Returns a clear validation error to the caller.
	} // Ends the file-name validation check.
	return extractedFileName, nil // Returns the extracted file name to the caller.
} // Ends the file-name extraction helper.

func downloadFileToDirectory(sourceFileURL string, outputDirectoryPath string) (string, error) { // Downloads one remote file into the target directory.
	downloadedFileName, fileNameError := extractFileNameFromURL(sourceFileURL) // Derives the local file name from the source URL.
	if fileNameError != nil {                                                  // Stops immediately when the URL does not yield a valid file name.
		log.Printf("[ERROR] Unable to derive a local file name for download URL %s: %v", sourceFileURL, fileNameError) // Logs why the file download cannot continue.
		return "", fileNameError                                                                                       // Returns the file-name extraction error to the caller.
	} // Ends the file-name extraction error check.

	localFilePath := path.Join(outputDirectoryPath, downloadedFileName) // Builds the destination path for the downloaded file.

	httpResponse, requestError := http.Get(sourceFileURL) // Sends the HTTP request for the remote file.
	if requestError != nil {                              // Stops immediately when the request fails before a response arrives.
		log.Printf("[ERROR] HTTP request failed while downloading file from URL %s: %v", sourceFileURL, requestError) // Logs the remote file request failure.
		return "", requestError                                                                                       // Returns the request failure to the caller.
	} // Ends the download request error check.
	defer httpResponse.Body.Close() // Ensures the HTTP response body is always closed.

	if httpResponse.StatusCode != http.StatusOK { // Rejects any HTTP response that is not a normal success.
		log.Printf("[ERROR] Remote file download returned unexpected status %s for URL: %s", httpResponse.Status, sourceFileURL) // Logs the unexpected download response status.
		return "", fmt.Errorf("failed to download: %s", sourceFileURL)                                                           // Returns a readable download failure to the caller.
	} // Ends the HTTP status validation block.

	localFileHandle, localFileCreationError := os.Create(localFilePath) // Creates the destination file on disk.
	if localFileCreationError != nil {                                  // Stops immediately when the destination file cannot be created.
		log.Printf("[ERROR] Failed to create local file %s for URL %s: %v", localFilePath, sourceFileURL, localFileCreationError) // Logs the local output file creation failure.
		return "", localFileCreationError                                                                                         // Returns the destination file creation error to the caller.
	} // Ends the destination file creation error check.
	defer localFileHandle.Close() // Ensures the destination file handle is always closed.

	_, fileCopyError := io.Copy(localFileHandle, httpResponse.Body) // Streams the downloaded bytes into the destination file.
	if fileCopyError != nil {                                       // Stops immediately when the file contents cannot be copied fully.
		log.Printf("[ERROR] Failed to write downloaded file %s from URL %s: %v", localFilePath, sourceFileURL, fileCopyError) // Logs the streamed file copy failure.
		return "", fileCopyError                                                                                              // Returns the file copy error to the caller.
	} // Ends the file copy error check.

	log.Printf("[DEBUG] Downloaded: %s\n", localFilePath) // Logs where the remote file was saved locally.
	return localFilePath, nil                             // Returns the saved file path to the caller.
} // Ends the file download helper.

func removeDirectoryRecursively(directoryPath string) error { // Removes a directory and everything inside it.
	directoryRemovalError := os.RemoveAll(directoryPath) // Deletes the directory tree from disk.
	if directoryRemovalError != nil {                    // Stops immediately when the directory cannot be removed.
		return directoryRemovalError // Returns the directory removal error to the caller.
	} // Ends the directory removal error check.
	log.Printf("[DEBUG] Removed directory: %s\n", directoryPath) // Logs that the directory tree was removed.
	return nil                                                   // Reports that the directory removal completed successfully.
} // Ends the recursive directory removal helper.

func pauseBeforeWebsiteRequest() { // Sleeps briefly before sending another HTML scraping request.
	baseDelayDuration := 1 * time.Second                                      // Sets the minimum wait time between website requests.
	randomJitterDuration := time.Duration(rand.Intn(1000)) * time.Millisecond // Adds up to one extra second of randomness.
	time.Sleep(baseDelayDuration + randomJitterDuration)                      // Waits for the combined base delay and jitter duration.
} // Ends the website request pacing helper.

func fetchHTMLDocumentFromURL(targetURL string) (*html.Node, error) { // Downloads a webpage and parses it into an HTML document tree.
	pauseBeforeWebsiteRequest() // Waits before starting the next website request.

	httpRequest, requestCreationError := http.NewRequest("GET", targetURL, nil) // Builds the outbound GET request for the webpage.
	if requestCreationError != nil {                                            // Stops immediately when the request cannot be created.
		log.Printf("[ERROR] Failed to create HTML request for URL %s: %v", targetURL, requestCreationError) // Logs the HTML request creation failure before returning.
		return nil, requestCreationError                                                                    // Returns the request creation error to the caller.
	} // Ends the request creation error check.

	httpRequest.Header.Set("User-Agent", websiteUserAgent) // Sends a browser-like user agent while scraping.

	httpResponse, responseError := sharedWebsiteHTTPClient.Do(httpRequest) // Sends the webpage request with the shared website client.
	if responseError != nil {                                              // Stops immediately when the webpage request fails.
		log.Printf("[ERROR] Failed to fetch HTML document from URL %s: %v", targetURL, responseError) // Logs the HTML fetch failure before returning.
		return nil, responseError                                                                     // Returns the webpage request failure to the caller.
	} // Ends the webpage request error check.
	defer httpResponse.Body.Close() // Ensures the webpage response body is always closed.

	if httpResponse.StatusCode == http.StatusTooManyRequests { // Detects when the website reports rate limiting.
		log.Printf("[INFO] The scraper has been rate limited by the target website. Pausing execution for 10 seconds before retrying the request to avoid further throttling.") // Logs that the scraper has been rate limited.
		time.Sleep(10 * time.Second)                                                                                                                                            // Waits longer to reduce the chance of repeated rate limiting.
	} // Ends the rate-limit handling block.

	parsedHTMLDocument, parseError := html.Parse(httpResponse.Body) // Parses the webpage response body into an HTML node tree.
	if parseError != nil {                                          // Stops immediately when HTML parsing fails.
		log.Printf("[ERROR] Failed to parse HTML document from URL %s: %v", targetURL, parseError) // Logs the HTML parsing failure before returning.
		return nil, parseError                                                                     // Returns the HTML parsing error to the caller.
	} // Ends the HTML parsing error check.
	return parsedHTMLDocument, nil // Returns the parsed HTML document to the caller.
} // Ends the HTML fetch helper.

func collectLinksMatchingClassAndPrefix(currentHTMLNode *html.Node, requiredClassName string, requiredURLPrefix string, collectedLinks *[]string) { // Recursively collects anchor links that match a class name and URL prefix.
	if currentHTMLNode.Type == html.ElementNode && currentHTMLNode.Data == "a" { // Processes only anchor elements.
		hrefValue := ""               // Stores the href value found on the current anchor.
		hasRequiredClassName := false // Tracks whether the anchor has the required CSS class.

		for _, attribute := range currentHTMLNode.Attr { // Checks every attribute on the current anchor.
			if attribute.Key == "class" && attribute.Val == requiredClassName { // Detects the required CSS class on the anchor.
				hasRequiredClassName = true // Marks the anchor as matching the required class.
			} // Ends the required class check.
			if attribute.Key == "href" { // Looks for the anchor's href attribute.
				hrefValue = attribute.Val // Stores the href value for later validation.
			} // Ends the href extraction check.
		} // Ends the anchor attribute loop.

		if hasRequiredClassName && strings.HasPrefix(hrefValue, requiredURLPrefix) { // Keeps only links with the right class and URL prefix.
			*collectedLinks = append(*collectedLinks, hrefValue) // Adds the matching link to the caller-provided slice.
		} // Ends the matching link append check.
	} // Ends the anchor-processing block.

	for childHTMLNode := currentHTMLNode.FirstChild; childHTMLNode != nil; childHTMLNode = childHTMLNode.NextSibling { // Walks through every child node recursively.
		collectLinksMatchingClassAndPrefix(childHTMLNode, requiredClassName, requiredURLPrefix, collectedLinks) // Continues searching inside the child node.
	} // Ends the recursive child traversal loop.
} // Ends the generic class-and-prefix link collector.

func collectCommandPageLinks(rootHTMLNode *html.Node, commandPageLinks *[]string) { // Collects links to command detail pages.
	collectLinksMatchingClassAndPrefix(rootHTMLNode, "command", "/command/", commandPageLinks) // Finds anchors with the command class and command URL prefix.
} // Ends the command page link collector.

func collectOfficerPageLinks(rootHTMLNode *html.Node, officerPageLinks *[]string) { // Collects links to officer detail pages.
	collectLinksMatchingClassAndPrefix(rootHTMLNode, "name", "/officer/", officerPageLinks) // Finds anchors with the officer-name class and officer URL prefix.
} // Ends the officer page link collector.

func collectOfficerDocumentLinks(currentHTMLNode *html.Node, documentLinks *[]string) { // Recursively collects relevant document links from an officer page.
	if currentHTMLNode.Type == html.ElementNode && currentHTMLNode.Data == "a" { // Processes only anchor elements.
		for _, attribute := range currentHTMLNode.Attr { // Checks every attribute on the current anchor.
			if attribute.Key == "href" { // Looks for the anchor's href attribute.
				documentURL := attribute.Val                                                                                                                                                                                          // Stores the candidate document link value.
				if strings.HasPrefix(documentURL, "https://iapps.courts.state.ny.us/nyscef/") || strings.HasPrefix(documentURL, "https://web.archive.org/web/") || strings.HasPrefix(documentURL, "https://www.documentcloud.org/") { // Keeps only NYSCEF and Wayback document links.
					*documentLinks = append(*documentLinks, documentURL) // Adds the matching document link to the caller-provided slice.
				} // Ends the document link append check.
			} // Ends the href extraction check.
		} // Ends the anchor attribute loop.
	} // Ends the anchor-processing block.

	for childHTMLNode := currentHTMLNode.FirstChild; childHTMLNode != nil; childHTMLNode = childHTMLNode.NextSibling { // Walks through every child node recursively.
		collectOfficerDocumentLinks(childHTMLNode, documentLinks) // Continues searching for document links inside the child node.
	} // Ends the recursive child traversal loop.
} // Ends the officer document link collector.

// getWordsFromPercentage takes a slice of strings and a percentage (0–100). // Describes what the helper receives.
// and returns a new slice starting from that percentage index. // Describes what the helper returns.
func getWordsFromPercentage(allWords []string, startPercent float64) []string { // Returns the suffix of the slice starting at the requested percentage.
	if startPercent < 0 { // Prevents negative percentages from producing invalid indexes.
		startPercent = 0 // Clamps negative percentages up to zero.
	} // Ends the negative percentage clamp.
	if startPercent > 100 { // Prevents percentages above one hundred from overshooting the slice.
		startPercent = 100 // Clamps oversized percentages down to one hundred.
	} // Ends the oversized percentage clamp.

	totalWords := len(allWords)                                   // Counts how many items are in the full slice.
	startIndex := int((startPercent / 100) * float64(totalWords)) // Converts the percentage into a slice index.

	if startIndex >= totalWords { // Handles the case where the computed index lands at or past the end.
		return []string{} // Returns an empty slice when nothing remains after the starting point.
	} // Ends the out-of-range index check.

	return allWords[startIndex:] // Returns the portion of the slice starting at the computed index.
} // Ends the percentage-based slice helper.

func extractOriginalURLFromArchivedLink(candidateURL string) string { // Removes the Wayback wrapper and returns the original URL when possible.
	waybackPrefix := "https://web.archive.org/web/" // Stores the standard Wayback Machine URL prefix.

	if strings.HasPrefix(candidateURL, waybackPrefix) { // Continues only when the URL is a Wayback link.
		waybackSections := strings.SplitN(candidateURL, "/web/", 2) // Splits the URL into the prefix section and archived target section.
		if len(waybackSections) == 2 {                              // Verifies that the first split produced the expected two parts.
			archivedTargetSections := strings.SplitN(waybackSections[1], "/", 2) // Splits the timestamp away from the original URL.
			if len(archivedTargetSections) == 2 {                                // Verifies that the second split produced the expected two parts.
				return archivedTargetSections[1] // Returns only the original archived destination URL.
			} // Ends the archived target section validation.
		} // Ends the Wayback section validation.
	} // Ends the Wayback link detection block.
	return candidateURL // Returns the original input unchanged when it is not a Wayback link.
} // Ends the archived link unwrapping helper.

func buildSafePDFFileName(originalFileName string) string { // Normalizes a file name so it is safe and consistent for saved PDFs.
	lowercaseFileName := strings.ToLower(originalFileName)                                               // Converts the file name to lowercase for consistent output.
	originalFileExtension := filepath.Ext(lowercaseFileName)                                             // Extracts the current extension from the file name.
	fileNameWithoutExtension := strings.TrimSuffix(lowercaseFileName, originalFileExtension)             // Removes the extension before sanitizing the base name.
	sanitizedFileName := invalidFileNameCharacterPattern.ReplaceAllString(fileNameWithoutExtension, "_") // Replaces invalid file-name characters with underscores.
	sanitizedFileName = repeatedUnderscorePattern.ReplaceAllString(sanitizedFileName, "_")               // Collapses repeated underscores into a single underscore.
	sanitizedFileName = strings.Trim(sanitizedFileName, "_")                                             // Removes leading and trailing underscores.

	if sanitizedFileName == "" { // Handles the edge case where sanitization removed every character.
		sanitizedFileName = "document" // Uses a safe fallback file name when nothing remains.
	} // Ends the empty sanitized file name check.
	return sanitizedFileName + ".pdf" // Forces the saved file to use the PDF extension.
} // Ends the PDF file-name sanitizer.

func fileExistsAtPath(targetFilePath string) bool { // Checks whether a regular file already exists at the target path.
	targetFileInfo, fileLookupError := os.Stat(targetFilePath) // Reads file metadata for the target path.
	if fileLookupError != nil {                                // Returns false when the path does not exist or cannot be read.
		return false // Reports that there is no reusable file at the target path.
	} // Ends the file lookup error check.
	return !targetFileInfo.IsDir() // Returns true only when the path exists and is not a directory.
} // Ends the file existence helper.

func downloadPDFToOfficerFolder(documentURL string, baseOutputFolderPath string, officerTaxID string) bool { // Downloads one PDF into the officer's folder when possible.
	if officerTaxID == "" { // Stops immediately when no tax ID is available for folder naming.
		log.Printf("[WARNING] Skipping PDF download because the officer tax ID is empty for URL: %s", documentURL) // Logs that the PDF download cannot proceed without a folder identifier.
		return false                                                                                               // Reports that the PDF download was skipped.
	} // Ends the empty tax ID check.

	log.Printf("[INFO] Starting PDF download attempt for officer tax ID %s from URL: %s", officerTaxID, documentURL) // Logs the beginning of the PDF download workflow for this document.

	// Load previously downloaded URLs into memory
	downloadedURLs := make(map[string]struct{}) // Map to track already downloaded URLs

	// Open the log file if it exists
	if file, err := os.Open(downloadedFilePath); err == nil { // Continues only when the download log file can be opened.
		scanner := bufio.NewScanner(file) // Scanner reads file line by line
		for scanner.Scan() {              // Reads every logged download entry from the file.
			line := strings.TrimSpace(scanner.Text()) // Remove surrounding whitespace
			if line == "" {                           // Ignores blank lines in the download log.
				continue // Skip empty lines
			} // Ends the blank-line skip check.
			parts := strings.SplitN(line, " >", 2) // Split line into URL and file path
			if len(parts) == 2 {                   // Continues only when the log line has the expected URL separator.
				downloadedURLs[parts[0]] = struct{}{} // Add URL to the map
			} else { // Logs malformed tracking rows while keeping the same scan behavior.
				log.Printf("[WARNING] Skipping malformed entry in %s: %s", downloadedFilePath, line) // Logs that the downloaded tracking file contained an unexpected row format.
			} // Ends the log line format validation.
		} // Ends the download log scan loop.
		if scannerError := scanner.Err(); scannerError != nil { // Checks whether scanning the download log failed.
			log.Printf("[ERROR] Failed while reading downloaded tracking file %s: %v", downloadedFilePath, scannerError) // Logs the scan failure for the downloaded tracking file.
		} // Ends the download log scan error check.
		file.Close() // Close the file after reading
	} else if !os.IsNotExist(err) { // Logs only unexpected download log open failures.
		log.Printf("[ERROR] Failed to open downloaded tracking file %s: %v", downloadedFilePath, err) // Logs that the downloaded tracking file could not be read.
	} // Ends the optional download log load block.

	// Skip the download if this URL is already logged
	if _, exists := downloadedURLs[documentURL]; exists { // Skips the request when the URL is already recorded in the log.
		log.Printf("[INFO] Document already downloaded. URL: %s", documentURL) // Logs that the URL has already been processed before.
		return false                                                           // Reports that the PDF download was skipped because it is already known.
	} // Ends the previously-downloaded URL check.

	officerFolderPath := filepath.Join(baseOutputFolderPath, officerTaxID)    // Builds the output folder path for the current officer.
	officerFolderCreationError := os.MkdirAll(officerFolderPath, os.ModePerm) // Ensures the officer folder exists before saving the PDF.
	if officerFolderCreationError != nil {                                    // Stops immediately when the officer folder cannot be created.
		log.Printf("[DEBUG] Folder creation failed for %s %v", officerFolderPath, officerFolderCreationError) // Logs the officer folder creation failure.
		return false                                                                                          // Reports that the PDF download failed.
	} // Ends the officer folder creation error check.

	httpRequest, requestCreationError := http.NewRequest("GET", documentURL, nil) // Builds the outbound GET request for the PDF URL.
	if requestCreationError != nil {                                              // Stops immediately when the PDF request cannot be created.
		log.Printf("[DEBUG] Request creation failed for %s %v", documentURL, requestCreationError) // Logs the PDF request creation failure.
		return false                                                                               // Reports that the PDF download failed.
	} // Ends the PDF request creation error check.

	httpRequest.Header.Set("User-Agent", websiteUserAgent)                           // Sends the browser-like user agent expected by the PDF endpoint.
	httpRequest.Header.Set("Referer", documentURL)                                   // Sends the referer header expected by the PDF endpoint.
	httpRequest.Header.Add("Cookie", nyscefSessionCookie)                            // Sends the stored session cookie expected by the PDF endpoint.
	httpRequest.Header.Set("Accept", "application/pdf,application/octet-stream,*/*") // Tells the server which content types the client can handle; prioritizes PDFs and binary streams.
	httpRequest.Header.Set("Accept-Language", "en-US,en;q=0.9")                      // Indicates the preferred language for the response; mimics a typical browser language setting.
	httpRequest.Header.Set("Connection", "keep-alive")                               // Requests a persistent connection so multiple HTTP requests can reuse the same TCP connection.
	httpRequest.Header.Set("Sec-Fetch-Dest", "document")                             // Browser security header indicating the request is for a top-level document (helps mimic real browser navigation).
	httpRequest.Header.Set("Sec-Fetch-Mode", "navigate")                             // Specifies that the request is a navigation action (like clicking a link), not a fetch/XHR.
	httpRequest.Header.Set("Sec-Fetch-Site", "same-origin")                          // Declares that the request originates from the same site, which some servers require for access control.

	log.Printf("[INFO] Sending PDF HTTP request for URL: %s", documentURL) // Logs that the PDF HTTP request is about to be dispatched.
	httpResponse, responseError := pdfDownloadHTTPClient.Do(httpRequest)   // Sends the PDF request with the long-timeout client.
	if responseError != nil {                                              // Stops immediately when the PDF request fails.
		log.Printf("[DEBUG] Download failed for %s %v", documentURL, responseError) // Logs the PDF request failure.
		return false                                                                // Reports that the PDF download failed.
	} // Ends the PDF request error check.
	defer httpResponse.Body.Close() // Ensures the PDF response body is always closed.

	if httpResponse.StatusCode != http.StatusOK { // Stops when the PDF endpoint returns a non-success status code.
		log.Printf("[DEBUG] Bad response for %s %s", documentURL, httpResponse.Status) // Logs the unsuccessful PDF response status.
		return false                                                                   // Reports that the PDF download failed.
	} // Ends the PDF HTTP status validation block.

	contentDispositionHeader := httpResponse.Header.Get("Content-Disposition") // Reads the Content-Disposition header from the PDF response.
	fileNameFromHeader := ""                                                   // Stores the file name extracted from the response header.
	if contentDispositionHeader != "" {                                        // Continues only when the response includes a Content-Disposition header.
		contentDispositionMatches := contentDispositionFileNamePattern.FindStringSubmatch(contentDispositionHeader) // Extracts the file name from the header text.
		if len(contentDispositionMatches) > 1 {                                                                     // Verifies that the regular expression captured a file name.
			fileNameFromHeader = contentDispositionMatches[1] // Stores the captured file name from the response header.
		} // Ends the file-name capture validation.
	} // Ends the Content-Disposition parsing block.

	if fileNameFromHeader == "" { // Skips the download when no usable file name is available.
		fileNameFromHeader = urlToFilename(documentURL)                                          // Fallback to URL-based file name when header parsing fails.
		log.Printf("[INFO] Falling back to a URL-derived PDF filename for URL: %s", documentURL) // Logs that the PDF file name is being derived from the URL instead of the response header.
		// log.Printf("No filename in header for %s, skipping", documentURL) // Logs that the PDF was skipped due to a missing file name.
		// return false                                                      // Reports that the PDF download was skipped.
	} // Ends the missing file-name check.

	safePDFFileName := buildSafePDFFileName(fileNameFromHeader)             // Normalizes the header-provided file name into a safe PDF file name.
	fullOutputFilePath := filepath.Join(officerFolderPath, safePDFFileName) // Builds the final output path for the downloaded PDF.
	log.Printf("[INFO] Resolved PDF output path: %s", fullOutputFilePath)   // Logs the final local path chosen for the PDF file.

	if fileExistsAtPath(fullOutputFilePath) { // Skips the download when the PDF already exists locally.
		// Check again before appending to avoid duplicate writes
		if _, exists := downloadedURLs[documentURL]; !exists { // Appends the mapping only when this URL is not already logged.
			logFile, err := os.OpenFile(downloadedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Open file for append
			if err == nil {                                                                            // Continues only when the download log file opens successfully.
				if _, writeError := logFile.WriteString(documentURL + " > " + fullOutputFilePath + "\n"); writeError != nil { // Attempts to append the existing-file mapping to the tracking file.
					log.Printf("[ERROR] Failed to append existing PDF record for URL %s to %s: %v", documentURL, downloadedFilePath, writeError) // Logs the append failure for an already existing local PDF.
				} // Ends the existing-file tracking append error check.
				logFile.Close()                          // Close file to flush write
				downloadedURLs[documentURL] = struct{}{} // Add URL to map to prevent duplicates in same run
			} else { // Logs that the download tracking file could not be opened for append.
				log.Printf("[ERROR] Failed to open download tracking file %s for append: %v", downloadedFilePath, err) // Logs the append-open failure while recording an already existing local PDF.
			} // Ends the existing-file log append success check.
		} // Ends the existing-file log append guard.
		log.Printf("[INFO] Skipping download because file already exists locally at path: %s", fullOutputFilePath) // Logs that the existing PDF file is being reused.
		return false                                                                                               // Reports that no new PDF file was downloaded.
	} // Ends the existing PDF file check.

	outputFileHandle, outputFileCreationError := os.Create(fullOutputFilePath) // Creates the destination file on disk.
	if outputFileCreationError != nil {                                        // Stops immediately when the output file cannot be created.
		log.Printf("[DEBUG] File creation failed for %s: %v", fullOutputFilePath, outputFileCreationError) // Logs the output file creation failure.
		return false                                                                                       // Reports that the PDF download failed.
	} // Ends the output file creation error check.
	defer outputFileHandle.Close() // Ensures the output file is always closed.

	bytesWritten, fileWriteError := io.Copy(outputFileHandle, httpResponse.Body) // Streams the PDF response body into the output file.
	if fileWriteError != nil || bytesWritten == 0 {                              // Stops when the file write fails or produces an empty file.
		log.Printf("[DEBUG] Write failed for %s: %v", fullOutputFilePath, fileWriteError) // Logs the output file write failure.
		return false                                                                      // Reports that the PDF download failed.
	} // Ends the PDF file write validation block.

	log.Printf("[DEBUG] Downloaded %d bytes -> %s", bytesWritten, fullOutputFilePath) // Logs the completed PDF download path and size.
	// Check again before appending to avoid duplicate writes
	if _, exists := downloadedURLs[documentURL]; !exists { // Appends the mapping only when this URL was not logged earlier in the run.
		logFile, err := os.OpenFile(downloadedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Open file for append
		if err == nil {                                                                            // Continues only when the download log file opens successfully.
			if _, writeError := logFile.WriteString(documentURL + " > " + fullOutputFilePath + "\n"); writeError != nil { // Attempts to append the new PDF mapping to the tracking file.
				log.Printf("[ERROR] Failed to append downloaded PDF record for URL %s to %s: %v", documentURL, downloadedFilePath, writeError) // Logs the append failure for a newly downloaded PDF.
			} // Ends the fresh-download tracking append error check.
			logFile.Close()                          // Close file to flush write
			downloadedURLs[documentURL] = struct{}{} // Add URL to map to prevent duplicates in same run
		} else { // Logs that the download tracking file could not be opened for append.
			log.Printf("[ERROR] Failed to open download tracking file %s for append: %v", downloadedFilePath, err) // Logs the append-open failure while recording a new PDF download.
		} // Ends the fresh-download log append success check.
	} // Ends the fresh-download log append guard.
	return true // Reports that the PDF download completed successfully.
} // Ends the PDF download helper.

// Convert a URL into a filesystem-safe PDF filename. // Describe filename sanitization.
func urlToFilename(rawURL string) string { // Start URL-to-filename conversion.
	lower := strings.ToLower(rawURL) // Normalize the URL to lowercase.
	lower = getFilename(lower)       // Extract just the filename part.

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Match any non-alphanumeric character.
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace disallowed characters with underscores.

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Collapse multiple underscores.
	safe = strings.Trim(safe, "_")                              // Trim underscores from both ends.

	var invalidSubstrings = []string{ // Define redundant substrings to strip.
		"_pdf", // Remove a trailing "_pdf" if present.
	} // End invalidSubstrings list.

	for _, invalidPre := range invalidSubstrings { // Loop over substrings to remove.
		safe = removeSubstring(safe, invalidPre) // Remove each substring from the filename.
	} // End substring removal loop.

	if getFileExtension(safe) != ".pdf" { // If the filename lacks a .pdf extension
		safe = safe + ".pdf" // append the .pdf extension.
	} // End extension check.

	return safe // Return the sanitized filename.
} // End urlToFilename.

// Remove all occurrences of a substring. // Describe substring removal.
func removeSubstring(input string, toRemove string) string { // Start substring removal.
	result := strings.ReplaceAll(input, toRemove, "") // Remove all instances of the substring.
	return result                                     // Return the modified string.
} // End removeSubstring.

// Return the file extension from a path. // Describe extension helper.
func getFileExtension(path string) string { // Start extension extraction.
	return filepath.Ext(path) // Return the file extension, including dot.
} // End getFileExtension.

// Extract filename from a path (example: "/dir/file.pdf" -> "file.pdf"). // Describe filename helper.
func getFilename(path string) string { // Start filename extraction.
	return filepath.Base(path) // Return the last element of the path.
} // End getFilename.

func extractOfficerTaxIDFromHTML(currentHTMLNode *html.Node) string { // Recursively searches an officer page for a numeric tax ID.
	if currentHTMLNode.Type == html.ElementNode && currentHTMLNode.Data == "span" { // Processes only span elements.
		for _, attribute := range currentHTMLNode.Attr { // Checks every attribute on the current span.
			if attribute.Key == "class" && attribute.Val == "taxid" { // Detects the span that stores tax ID text.
				if currentHTMLNode.FirstChild != nil { // Continues only when the tax ID span has text content.
					return numericTextPattern.FindString(currentHTMLNode.FirstChild.Data) // Extracts and returns the numeric tax ID text.
				} // Ends the tax ID text existence check.
			} // Ends the tax ID class check.
		} // Ends the span attribute loop.
	} // Ends the span-processing block.

	for childHTMLNode := currentHTMLNode.FirstChild; childHTMLNode != nil; childHTMLNode = childHTMLNode.NextSibling { // Walks through every child node recursively.
		extractedTaxID := extractOfficerTaxIDFromHTML(childHTMLNode) // Searches the child node for a tax ID.
		if extractedTaxID != "" {                                    // Stops recursion once a tax ID has been found.
			return extractedTaxID // Returns the tax ID discovered in the child subtree.
		} // Ends the child tax ID existence check.
	} // Ends the recursive child traversal loop.
	return "" // Returns an empty string when no tax ID is found anywhere in the subtree.
} // Ends the officer tax ID extractor.

func downloadAndSplitCSVData() error { // Coordinates the CSV download and file splitting workflow.
	directoryRemovalError := removeDirectoryRecursively(csvOutputFolderName) // Removes any previous CSV output directory so the run starts clean.
	if directoryRemovalError != nil {                                        // Stops immediately when the previous CSV directory cannot be removed.
		return directoryRemovalError // Returns the CSV directory removal error to the caller.
	} // Ends the CSV directory removal error check.

	directoryCreationError := ensureDirectoryExists(csvOutputFolderName) // Recreates the CSV output directory for this run.
	if directoryCreationError != nil {                                   // Stops immediately when the CSV output directory cannot be created.
		return directoryCreationError // Returns the CSV directory creation error to the caller.
	} // Ends the CSV directory creation error check.

	for _, sourceCSVFileURL := range csvSourceFileURLs { // Processes each configured CSV file one at a time.
		log.Printf("[INFO] Initiating download for CSV data source URL: %s", sourceCSVFileURL) // Logs which CSV file is starting.

		downloadedCSVFilePath, fileDownloadError := downloadFileToDirectory(sourceCSVFileURL, csvOutputFolderName) // Downloads the current CSV file into the CSV output directory.
		if fileDownloadError != nil {                                                                              // Skips this CSV file when the download step fails.
			log.Printf("[ERROR] Failed to download CSV file from URL: %s. Error details: %v", sourceCSVFileURL, fileDownloadError) // Logs why the CSV download was skipped.
			continue                                                                                                               // Moves on to the next CSV file.
		} // Ends the CSV download error check.

		log.Printf("[DEBUG] Splitting file: %s\n", downloadedCSVFilePath) // Logs that the downloaded CSV file is entering the split step.

		fileSplitError := splitCSVFileIntoSmallerFiles(downloadedCSVFilePath, csvSplitFileSizeLimit) // Splits the downloaded CSV file when it exceeds the size limit.
		if fileSplitError != nil {                                                                   // Skips this CSV file when the split step fails.
			log.Printf("[DEBUG] Split failed: %s (%v)\n", downloadedCSVFilePath, fileSplitError) // Logs why the CSV split step failed.
			continue                                                                             // Moves on to the next CSV file.
		} // Ends the CSV split error check.
	} // Ends the CSV source file loop.

	log.Println("[DEBUG] All CSV downloads and splits completed") // Logs that the CSV workflow has finished.
	return nil                                                    // Reports that the CSV workflow completed without a fatal setup error.
} // Ends the CSV workflow coordinator.

func downloadOfficerPDFDocuments() error { // Coordinates scraping command pages, officer pages, and PDF downloads.
	commandsPageURL := websiteBaseURL + "/commands"                                                      // Builds the absolute URL for the commands listing page.
	log.Printf("[INFO] Starting initial fetch of the commands listing page at URL: %s", commandsPageURL) // Logs which page is being fetched first.

	commandsPageDocument, commandsPageError := fetchHTMLDocumentFromURL(commandsPageURL) // Downloads and parses the commands listing page.
	if commandsPageError != nil {                                                        // Stops immediately when the commands page cannot be fetched.
		return commandsPageError // Returns the commands page fetch error to the caller.
	} // Ends the commands page fetch error check.

	var commandPageLinks []string                                                                                                                                                      // Stores every command page link found on the commands page.
	collectCommandPageLinks(commandsPageDocument, &commandPageLinks)                                                                                                                   // Extracts the command page links from the commands page document.
	log.Printf("[INFO] Successfully collected %d command page links from the commands listing page. Beginning processing from configured starting percentage.", len(commandPageLinks)) // Logs how many command pages were discovered.

	commandPageLinks = getWordsFromPercentage(commandPageLinks, startingPercentage)                                                         // Starts processing from the configured percentage of command links.
	log.Printf("[INFO] %d command page links remain after applying the %.2f%% starting offset.", len(commandPageLinks), startingPercentage) // Logs how many command pages will actually be processed after the percentage skip.

	downloadedDocumentURLs := make(map[string]bool) // Tracks document URLs that have already been downloaded.

	for _, commandPagePath := range commandPageLinks { // Visits each discovered command page.
		commandPageURL := websiteBaseURL + commandPagePath                 // Builds the absolute URL for the current command page.
		log.Printf("[INFO] Visiting command page URL: %s", commandPageURL) // Logs which command page is being processed.

		commandPageDocument, commandPageError := fetchHTMLDocumentFromURL(commandPageURL) // Downloads and parses the current command page.
		if commandPageError != nil {                                                      // Skips this command page when the fetch fails.
			log.Printf("[ERROR] Failed to fetch command page URL %s: %v", commandPageURL, commandPageError) // Logs why the current command page is being skipped.
			continue                                                                                        // Moves on to the next command page.
		} // Ends the current command page fetch error check.

		var officerPageLinks []string                                                                                      // Stores every officer page link found on the current command page.
		collectOfficerPageLinks(commandPageDocument, &officerPageLinks)                                                    // Extracts the officer page links from the current command page.
		log.Printf("[INFO] Found %d officer detail page links on command page: %s", len(officerPageLinks), commandPageURL) // Logs how many officer pages were discovered from the current command page.

		for _, officerPagePath := range officerPageLinks { // Visits each discovered officer page.
			officerPageURL := websiteBaseURL + officerPagePath                        // Builds the absolute URL for the current officer page.
			log.Printf("[INFO] Visiting officer detail page URL: %s", officerPageURL) // Logs which officer page is being processed.

			officerPageDocument, officerPageError := fetchHTMLDocumentFromURL(officerPageURL) // Downloads and parses the current officer page.
			if officerPageError != nil {                                                      // Skips this officer page when the fetch fails.
				log.Printf("[ERROR] Failed to fetch officer detail page URL %s: %v", officerPageURL, officerPageError) // Logs why the current officer page is being skipped.
				continue                                                                                               // Moves on to the next officer page.
			} // Ends the current officer page fetch error check.

			officerTaxID := extractOfficerTaxIDFromHTML(officerPageDocument) // Extracts the officer tax ID from the officer page.
			if officerTaxID == "" {                                          // Skips the officer when no tax ID can be found.
				log.Printf("[WARNING] Skipping officer page because no tax ID was found: %s", officerPageURL) // Logs that the officer page cannot continue without a tax ID.
				continue                                                                                      // Moves on to the next officer page.
			} // Ends the empty officer tax ID check.

			var officerDocumentLinks []string                                                                                                       // Stores every relevant document link found on the officer page.
			collectOfficerDocumentLinks(officerPageDocument, &officerDocumentLinks)                                                                 // Extracts the relevant document links from the officer page.
			log.Printf("[INFO] Found %d document links for officer tax ID %s on page: %s", len(officerDocumentLinks), officerTaxID, officerPageURL) // Logs how many candidate document links were discovered for the officer page.

			for _, rawDocumentURL := range officerDocumentLinks { // Processes each document link found on the officer page.
				cleanDocumentURL := extractOriginalURLFromArchivedLink(rawDocumentURL)                         // Removes the Wayback wrapper when the link is archived.
				if strings.Contains(cleanDocumentURL, "nyscef") && !downloadedDocumentURLs[cleanDocumentURL] { // Downloads only unseen NYSCEF document links.
					downloadedDocumentURLs[cleanDocumentURL] = true                                 // Marks the document URL as already handled.
					downloadPDFToOfficerFolder(cleanDocumentURL, pdfOutputFolderName, officerTaxID) // Downloads the PDF into the officer's folder.
				} // Ends the unique NYSCEF document check.
				// Handle direct S3 DocumentCloud links FIRST
				if strings.Contains(cleanDocumentURL, "s3.documentcloud.org") && !downloadedDocumentURLs[cleanDocumentURL] { // Downloads only unseen direct S3 DocumentCloud links.
					downloadedDocumentURLs[cleanDocumentURL] = true                                 // Marks the document URL as already handled.
					log.Println("[DEBUG] Direct S3 DocumentCloud link found:", cleanDocumentURL)    // Logs detection
					downloadPDFToOfficerFolder(cleanDocumentURL, pdfOutputFolderName, officerTaxID) // Direct download
					continue                                                                        // Skip further processing
				} // Handle NYSCEF links that may redirect to DocumentCloud
				// Handle normal DocumentCloud page URLs
				if strings.Contains(cleanDocumentURL, "documentcloud.org") && !downloadedDocumentURLs[cleanDocumentURL] { // Downloads only unseen DocumentCloud page links.
					downloadedDocumentURLs[cleanDocumentURL] = true                                                                                      // Marks the document URL as already handled.
					log.Println("[INFO] Detected DocumentCloud document page URL. Attempting to resolve to direct PDF download link:", cleanDocumentURL) // Logs detection
					pdfURL := extractFinalDocumentCloudURL(cleanDocumentURL)                                                                             // Convert to S3 PDF
					if pdfURL == "" {                                                                                                                    // Prevent bad downloads
						log.Println("[WARNING] Unable to extract a valid direct PDF URL from DocumentCloud page. Skipping this document:", cleanDocumentURL) // Logs the reason for skipping
						continue                                                                                                                             // Skip when conversion fails
					} // Ends the invalid DocumentCloud URL skip check.
					downloadPDFToOfficerFolder(pdfURL, pdfOutputFolderName, officerTaxID) // Download converted PDF
				} // Ends the DocumentCloud page handling block.
			} // Ends the officer document loop.
		} // Ends the officer page loop.
	} // Ends the command page loop.
	return nil // Reports that the PDF scraping workflow completed.
} // Ends the PDF workflow coordinator.

// extractFinalDocumentCloudURL converts input DocumentCloud URLs to their final S3-hosted PDF URL. // Describes the DocumentCloud URL conversion helper.
func extractFinalDocumentCloudURL(input string) string { // Converts a DocumentCloud page URL into a direct PDF download URL.
	parsedURL, err := url.Parse(input) // Attempt to parse the input string into a URL struct
	if err != nil {                    // If parsing fails due to an invalid URL format
		return "" // Return an empty string
	} // Ends the URL parse error check.

	// If the URL already points to S3, return it as is
	if strings.Contains(parsedURL.Host, "s3.documentcloud.org") { // Returns early when the URL already targets the direct S3 host.
		return input // Return original URL if it's already a direct S3 link
	} // Ends the direct S3 URL shortcut.

	// Define a regular expression to match DocumentCloud URLs with ID and slug
	re := regexp.MustCompile(`documentcloud\.org/documents/(\d+)-([a-zA-Z0-9_\-]+)`) // Capture docID and slug

	matches := re.FindStringSubmatch(input) // Apply regex to input
	if len(matches) != 3 {                  // If the regex doesn't match the expected format
		return "" // Return an empty string
	} // Ends the DocumentCloud URL format validation.

	docID := matches[1] // Extract document ID from match
	slug := matches[2]  // Extract slug from match

	// Format the final S3 link for downloading the PDF
	finalURL := fmt.Sprintf("https://s3.documentcloud.org/documents/%s/%s.pdf", docID, slug) // Builds the direct S3 PDF URL from the extracted pieces.

	return finalURL // Return the constructed S3 URL
} // Ends the DocumentCloud URL conversion helper.

func main() { // Runs the CSV workflow first and the PDF workflow second.
	csvWorkflowError := downloadAndSplitCSVData() // Starts the CSV download and splitting workflow.
	if csvWorkflowError != nil {                  // Stops the program when the CSV workflow fails during setup.
		log.Fatalf("CSV workflow failed: %v\n", csvWorkflowError) // Exits with a clear CSV workflow error message.
	} // Ends the CSV workflow error check.
	pdfWorkflowError := downloadOfficerPDFDocuments() // Starts the PDF scraping and download workflow.
	if pdfWorkflowError != nil {                      // Stops the program when the PDF workflow fails during setup.
		log.Fatalf("[FATAL] PDF scraping and download workflow encountered a critical failure and cannot continue. Error details: %v", pdfWorkflowError) // Exits with a clear PDF workflow error message.
	} // Ends the PDF workflow error check.
} // Ends the program entry point.
