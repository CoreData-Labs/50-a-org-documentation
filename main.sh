#!/bin/bash

# -----------------------------------------------------------------------------
# Function: main
# Purpose : Automatically stage, commit, and push changes based on:
#           1) Minimum number of file changes (threshold trigger)
#           2) Maximum elapsed time since last push (time trigger)
# -----------------------------------------------------------------------------
function main() {

	# --- Configuration --------------------------------------------------------

	CHECK_INTERVAL_SECONDS=60     # How often to check repo status
	MIN_WAIT_SECONDS=3600         # Force push after 60 minutes
	MIN_FILE_CHANGE_THRESHOLD=500 # Push early if >= this many file changes

	# Track last successful push time (epoch seconds)
	last_push_epoch=$(date +%s)

	# --- Continuous Monitoring Loop ------------------------------------------

	while true; do

		current_epoch=$(date +%s)
		elapsed_seconds=$((current_epoch - last_push_epoch))

		# Count changed files (staged + unstaged + untracked)
		changed_files_count=$(git status --porcelain 2>/dev/null | wc -l)

		# --- Status Output ----------------------------------------------------

		echo "================================================================"
		echo "🕒 Timestamp           : $(date)"
		echo "📁 Changed Files       : ${changed_files_count}"
		echo "⏳ Time Since Last Push: ${elapsed_seconds}s"
		echo "================================================================"

		# --- Trigger Condition ------------------------------------------------
		# Push if:
		#   A) File changes exceed threshold
		#   B) Max wait time exceeded
		if [[ ${changed_files_count} -ge ${MIN_FILE_CHANGE_THRESHOLD} || ${elapsed_seconds} -ge ${MIN_WAIT_SECONDS} ]]; then

			# No-op protection: avoid empty commits
			if [[ ${changed_files_count} -eq 0 ]]; then
				echo "⚠️  Trigger reached, but no changes detected. Resetting timer."
				last_push_epoch=$current_epoch
				sleep "${CHECK_INTERVAL_SECONDS}"
				continue
			fi

			# --- Sync with Remote --------------------------------------------

			echo "🔄 Syncing with remote (git pull --rebase --autostash)..."
			if ! git pull --rebase --autostash; then
				echo "❌ ERROR: Failed to pull/rebase from remote."
				echo "   ➤ Possible causes:"
				echo "     - Merge conflicts requiring manual resolution"
				echo "     - Network connectivity issues"
				echo "     - Invalid git remote configuration"
				echo "   ➤ Action: Resolve manually, then rerun script."
				sleep "${CHECK_INTERVAL_SECONDS}"
				continue
			fi

			# --- Stage Changes ------------------------------------------------

			echo "➕ Staging all changes (including deletions)..."
			if ! git add -A; then
				echo "❌ ERROR: Failed to stage changes."
				echo "   ➤ Check file permissions or repository integrity."
				sleep "${CHECK_INTERVAL_SECONDS}"
				continue
			fi

			# --- Commit Changes -----------------------------------------------

			commit_timestamp=$(date -u +'%Y-%m-%d %H:%M:%S UTC')
			commit_message="🤖 Auto Sync: ${commit_timestamp}"

			echo "📝 Creating commit..."
			if git commit -m "${commit_message}"; then
				echo "✅ Commit created successfully."
			else
				echo "⚠️  No changes to commit after staging (possibly already committed)."
				sleep "${CHECK_INTERVAL_SECONDS}"
				continue
			fi

			# --- Push to Remote -----------------------------------------------

			echo "🚀 Pushing changes to remote repository..."
			if git push; then
				echo "✅ Push completed successfully."

				# Reset timer after successful push
				last_push_epoch=$current_epoch
			else
				echo "❌ ERROR: Failed to push changes."
				echo "   ➤ Possible causes:"
				echo "     - Authentication failure (SSH/HTTPS credentials)"
				echo "     - Remote rejected push (e.g., protected branch)"
				echo "     - Network issues"
				echo "   ➤ Action: Verify credentials and remote permissions."
			fi
		fi

		# --- Wait Before Next Cycle -------------------------------------------

		sleep "${CHECK_INTERVAL_SECONDS}"
	done
}

# -----------------------------------------------------------------------------
# Entry Point
# -----------------------------------------------------------------------------

main
