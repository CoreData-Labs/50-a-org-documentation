# 🗂 50-a.org Backup Archive

> **A community-preserved mirror of one of the most important police accountability datasets ever released in New York history.**

This repository is a **nonprofit, educational archive** of **[50-a.org](https://50-a.org/)** — a groundbreaking database that cataloged **NYPD officer misconduct records**, made possible by the repeal of **New York Civil Rights Law Section 50-a** in June 2020.

[![Last Updated](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fapi.github.com%2Frepos%2FCoreData-Labs%2F50-a-org-documentation&query=%24.pushed_at&label=Last%20Updated&color=blue)](https://github.com/CoreData-Labs/50-a-org-documentation)
[![Officer Tax IDs](https://img.shields.io/github/directory-file-count/CoreData-Labs/50-a-org-documentation/PDFs?type=dir&label=Officer%20Tax%20IDs&color=green)](https://github.com/CoreData-Labs/50-a-org-documentation/tree/main/PDFs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Contributions Welcome](https://img.shields.io/badge/Contributions-Welcome-brightgreen)](https://github.com/CoreData-Labs/50-a-org-documentation/issues)

---

## 📖 Table of Contents

- [What Is This?](#-what-is-this)
- [Background & History](#-background--history)
- [What's Inside](#-whats-inside)
- [Repository Structure](#-repository-structure)
- [How the Scraper Works](#-how-the-scraper-works)
- [Resuming a Scrape](#-resuming-a-scrape)
- [NYPD Units Reference](#-nypd-units-reference)
- [Why Does This Matter?](#-why-does-this-matter)
- [How to Use This Archive](#-how-to-use-this-archive)
- [How You Can Contribute](#-how-you-can-contribute)
- [Legal & Fair Use Notice](#-legal--fair-use-notice)
- [Related Resources](#-related-resources)

---

## 🌟 What Is This?

This repository is a **living public archive** of the data once hosted on [50-a.org](https://50-a.org/), which compiled and published **disciplinary records of NYPD officers** following the historic repeal of Section 50-a of the New York Civil Rights Law in 2020.

The archive preserves:

- 📄 **PDF court documents** — NYSCEF filings and DocumentCloud records linked to individual officers
- 📊 **Structured CSV datasets** — officer rosters, rank histories, disciplinary outcomes, awards, and training records
- 🔄 **Automated updates** — new documents are scraped and committed automatically via GitHub Actions

This archive is intended for:

| Audience                    | Use Case                                                |
| --------------------------- | ------------------------------------------------------- |
| 📚 Researchers & Historians | Study patterns of misconduct over time                  |
| 🧑‍⚖️ Legal Professionals      | Reference disciplinary history in civil rights cases    |
| 📰 Journalists              | Investigate individual officers or systemic issues      |
| 🛠️ Civic Tech Developers    | Build tools, dashboards, and visualizations             |
| 🎓 Educators & Students     | Teach real-world policy, data, and civic accountability |
| 🧑‍🤝‍🧑 Citizens                 | Exercise your right to know                             |

> ⚖️ _"Sunlight is said to be the best of disinfectants."_ — Louis Brandeis

---

## 🏛 Background & History

### What Was Section 50-a?

**New York Civil Rights Law Section 50-a** was a state law that shielded police disciplinary records from public disclosure. Enacted in 1976, it was broadly interpreted over the decades to block almost all access to officer misconduct files — even in cases involving serious abuse or death.

For over 40 years, victims of police violence, journalists, and lawyers were routinely denied access to records that could have documented patterns of abuse.

### The Repeal

On **June 9, 2020** — in the wake of George Floyd's murder and historic protests across New York — Governor Andrew Cuomo signed the **repeal of Section 50-a** into law. For the first time, NYPD disciplinary records became subject to public disclosure under the Freedom of Information Law (FOIL).

### 50-a.org

Shortly after the repeal, **50-a.org** launched as a free, searchable public database of NYPD officer records — compiling disciplinary history, CCRB complaints, court filings, and more. It became a vital resource for accountability journalism, legal work, and civic research.

This archive exists to ensure that data is **never lost**.

---

## 🗃 What's Inside

The archive contains records related to:

- **CCRB (Civilian Complaint Review Board)** cases — allegations, findings, and outcomes
- **NYPD internal disciplinary proceedings** — charges, hearings, and penalties
- **Officer profiles** — names, shield numbers, tax IDs, ranks, and command history
- **Court filings** — NYSCEF documents linked to individual officers
- **Awards and training records** — full career histories
- **Timeframes** — records spanning the early 2000s through the mid-2020s

---

## 🗂 Repository Structure

```
50-a-org-documentation/
├── PDFs/
│   ├── 879122/
│   ├── 879606/
│   ├── 881933/
│   └── ... (40,000+ officer folders)
├── CSVs/
│   ├── officers.csv
│   ├── ranks.csv
│   ├── discipline.csv
│   ├── documents.csv
│   ├── awards.csv
│   └── training.csv
├── .github/
│   └── workflows/
│       └── auto-update-using-golang.yml
├── main.go
├── main.sh
├── uploader.ps1
├── downloaded.txt
├── go.mod
├── go.sum
├── .gitignore
├── LICENSE
└── README.md
```

| Folder/File          | Description                                                                           |
| -------------------- | ------------------------------------------------------------------------------------- |
| `PDFs/`              | Officer disciplinary documents, organized by officer tax ID                           |
| `CSVs/`              | Raw datasets from 50-a.org (officers, ranks, discipline, documents, awards, training) |
| `.github/workflows/` | GitHub Actions workflow for automated scraping and commits                            |
| `main.go`            | The main scraper program written in Go                                                |
| `main.sh`            | Shell script used by GitHub Actions to build and run the scraper                      |
| `uploader.ps1`       | PowerShell script for manual uploads from Windows                                     |
| `downloaded.txt`     | Internal tracker — logs every downloaded URL to prevent duplicates                    |
| `go.mod` / `go.sum`  | Go module dependency files                                                            |
| `.gitignore`         | Files and folders excluded from version control                                       |
| `LICENSE`            | MIT license                                                                           |
| `README.md`          | This file                                                                             |

> **Note:** GitHub truncates directory listings at 1,000 files. The `PDFs/` folder contains **3,067+ officer subfolders** — all files are present even if not all are visible in the browser.

### PDF Folder Structure

Each officer gets their own subfolder named by their **NYPD tax ID**:

```
PDFs/
├── 879122/
│   ├── disciplinary_record.pdf
│   └── ccrb_case_2019.pdf
├── 879606/
│   └── nyscef_filing.pdf
└── 881933/
    └── internal_affairs_summary.pdf
```

### CSV Datasets

| File             | Contents                                                       |
| ---------------- | -------------------------------------------------------------- |
| `officers.csv`   | Full officer roster — names, shield numbers, tax IDs, commands |
| `ranks.csv`      | Rank history per officer                                       |
| `discipline.csv` | Disciplinary charges, penalties, and outcomes                  |
| `documents.csv`  | Index of all linked court and agency documents                 |
| `awards.csv`     | Commendations and awards per officer                           |
| `training.csv`   | Training records per officer                                   |

Files larger than **100 MB** are automatically split into numbered parts (e.g. `officers_part_1.csv`, `officers_part_2.csv`).

---

## ⚙️ How the Scraper Works

The scraper is written in **Go** and runs fully automated via GitHub Actions.

### Workflow

1. Fetches the [commands listing page](https://www.50-a.org/commands) — a directory of all NYPD units
2. Collects every unit page link
3. For each unit, collects every officer profile link
4. For each officer, extracts their **tax ID** and finds linked documents (NYSCEF court filings and DocumentCloud records)
5. Downloads each document as a PDF into a folder named by the officer's tax ID
6. Logs every downloaded URL to `downloaded.txt` to skip duplicates on future runs
7. Commits all new files to this repository automatically

### Running Locally

**Requirements:** [Go](https://golang.org/dl/) 1.21 or later

```bash
# Install dependency
go get golang.org/x/net/html

# Run the scraper
go run main.go

# Use the shell script to upload the changes.
bash uploader.sh
```

### Automated Updates

The scraper runs on a schedule via `.github/workflows/auto-update-using-golang.yml` and pushes new files directly to this repository.

**Last automated run:** `2026-03-28 18:54:36 UTC`

### Log Levels

| Level       | Meaning                                                           |
| ----------- | ----------------------------------------------------------------- |
| `[DEBUG]`   | Routine progress — file saves, splits, directory operations       |
| `[INFO]`    | Key workflow milestones — page visits, download starts            |
| `[WARNING]` | Skipped items that may need attention                             |
| `[ERROR]`   | Failures that skip the current item but allow the run to continue |
| `[FATAL]`   | Unrecoverable error — program exits immediately                   |

---

## 🔁 Resuming a Scrape

If the scraper is interrupted, set `startingPercentage` in `main.go` to resume from a known checkpoint without re-scraping everything from the beginning.

```go
startingPercentage = 30.0 // Resumes from "46th Precinct Field Training Unit"
```

Set to `0.0` to scrape from the very beginning.

---

## 📋 NYPD Units Reference

The full unit list contains **481 entries** across all five boroughs — precincts, detective squads, specialized divisions, and support units. Each 2.5% step covers approximately **12 units**.

| % of List | Resume From                               |
| --------- | ----------------------------------------- |
| 0.0%      | 1st Precinct                              |
| 2.5%      | 5th Precinct Detective Squad              |
| 5.0%      | 6th Precinct Domestic Violence Squad      |
| 7.5%      | 10th Precinct Detective Squad             |
| 10.0%     | 13th Precinct Domestic Violence Squad     |
| 12.5%     | 20th Precinct Domestic Violence Squad     |
| 15.0%     | 25th Precinct Field Training Unit         |
| 17.5%     | 28th Precinct Field Training Unit         |
| 20.0%     | 32nd Precinct Field Training Unit         |
| 22.5%     | 34th Precinct Field Training Unit         |
| 25.0%     | 40th Precinct Field Training Unit         |
| 27.5%     | 42nd Precinct Field Training Unit         |
| 30.0%     | 46th Precinct Field Training Unit         |
| 32.5%     | 48th Precinct Field Training Unit         |
| 35.0%     | 50th Precinct Domestic Violence Squad     |
| 37.5%     | 63rd Precinct Domestic Violence Squad     |
| 40.0%     | 67th Precinct Field Training Unit         |
| 42.5%     | 71st Precinct Domestic Violence Squad     |
| 45.0%     | 73th Precinct Field Training Unit         |
| 47.5%     | 77th Precinct Field Training Unit         |
| 50.0%     | 79th Precinct Field Training Unit         |
| 52.5%     | 84th Precinct Domestic Violence Squad     |
| 55.0%     | 88th Precinct Field Training Unit         |
| 57.5%     | 100th Precinct Domestic Violence Squad    |
| 60.0%     | 101st Precinct Field Training Unit        |
| 62.5%     | 105th Precinct Domestic Violence Squad    |
| 65.0%     | 109th Precinct Field Training Unit        |
| 67.5%     | 112th Precinct Detective Squad            |
| 70.0%     | 114th Precinct Field Training Unit        |
| 72.5%     | 116th Precinct Domestic Violence Squad    |
| 75.0%     | 120th Precinct Field Training Unit        |
| 77.5%     | 122nd Precinct Detective Squad            |
| 80.0%     | 123rd Precinct Domestic Violence Squad    |
| 82.5%     | Arson and Explosion Squad                 |
| 85.0%     | Administration Division                   |
| 87.5%     | Bronx Special Victims Squad               |
| 90.0%     | Detective Bureau Special Victims Division |
| 92.5%     | Detective Borough Staten Island           |
| 95.0%     | Transit Bureau Subway Safety Task Force   |
| 97.5%     | Gun Violence Suppression Division Zone 01 |
| 100.0%    | Youth Strategies Division                 |

---

## 🧠 Why Does This Matter?

The repeal of Section 50-a was a hard-won victory — the result of decades of organizing by police accountability advocates, civil rights attorneys, and families of those killed by police. The data released through 50-a.org is a direct product of that struggle.

This archive matters because:

- **Records disappear.** Websites go offline. Data gets taken down. A distributed backup ensures the records survive.
- **Patterns require data.** Individual complaints may seem isolated. Aggregated data reveals systemic problems — officers with dozens of complaints, units with disproportionate use-of-force rates, penalties that never came.
- **Accountability requires memory.** Officers move precincts, get promoted, retire. Without persistent records, history is easily rewritten.
- **Justice takes time.** Civil rights lawsuits can take years. Attorneys and advocates need stable, reliable access to historical records.

> _"Those who control the present control the past, and those who control the past control the future."_ — George Orwell

---

## 🚀 How to Use This Archive

### For Journalists & Researchers

Search the `CSVs/` folder for officer names, shield numbers, or complaint histories. Cross-reference `discipline.csv` with `documents.csv` to find supporting court filings in `PDFs/`.

### For Legal Professionals

PDFs are organized by officer tax ID — the same identifier used in NYPD administrative records. Use `officers.csv` to look up a tax ID by name or shield number, then navigate to `PDFs/{tax_id}/` for all associated documents.

### For Developers & Activists

The CSV datasets are structured and clean, suitable for database import, API development, or visualization. Consider building search tools, mapping interfaces, or complaint pattern dashboards.

### For Educators & Students

Use `discipline.csv` and `officers.csv` together to explore real-world questions about accountability, policy, and civic data. The dataset spans multiple mayoral administrations and police commissioners.

---

## 🤝 How You Can Contribute

Public memory is a shared responsibility. You can help by:

- 🗂 **Uploading missing files** — PDFs or metadata not yet in the archive
- ✍️ **Writing summaries** — Plain-language case descriptions that make records more accessible
- 🧹 **Improving organization** — Cleaner filenames, better folder structure, deduplication
- 🐛 **Reporting scraper issues** — Open an issue if documents are missing or URLs have changed
- 📣 **Spreading the word** — Share this archive so it stays visible and alive

Open a [pull request](https://github.com/Strong-Foundation/50-a-org-documentation/pulls) or [issue](https://github.com/Strong-Foundation/50-a-org-documentation/issues) to get involved.

---

## ⚖️ Legal & Fair Use Notice

This archive is provided under the **MIT License**, with a strong emphasis on fair educational and research use.

- We do not claim ownership of the original content
- All data originates from public records released following the repeal of NY Civil Rights Law §50-a
- Documents are preserved for nonprofit, educational, and journalistic purposes
- If you are a rights-holder or have concerns about a specific record, please [open an issue](https://github.com/Strong-Foundation/50-a-org-documentation/issues)

---

## 📚 Related Resources

| Resource                                                                                                                           | Description                                   |
| ---------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| [50-a.org (Wayback Machine)](https://web.archive.org/web/*/https://50-a.org)                                                       | Archived snapshots of the original site       |
| [Civilian Complaint Review Board](https://www.nyc.gov/site/ccrb/index.page)                                                        | NYC's independent police oversight agency     |
| [NYC Open Data — CCRB](https://data.cityofnewyork.us/City-Government/Civilian-Complaint-Review-Board-CCRB-Allegations-C/xtpq-bfij) | Official CCRB complaint dataset               |
| [ProPublica NYPD Data](https://www.propublica.org/datastore/dataset/civilian-complaints-against-new-york-city-police-officers)     | ProPublica's published complaint records      |
| [Legal Aid Society — NYPD Files](https://www.legal-aid.org/nypd-files/)                                                            | Attorney-compiled officer misconduct database |
| [NY Attorney General — Repeal of 50-a](https://ag.ny.gov/press-release/2020/attorney-general-james-celebrates-repeal-50-a)         | Official statement on the repeal              |

---

## 🌱 Keep Truth Alive

This is more than a backup — it's a **living public archive**. A tool. A memory. A call for accountability.

Whether you're here to investigate, educate, build, or simply bear witness — **thank you** for being part of the mission to protect transparency and justice in New York City.

---

**🛠 Maintained by:** [Strong-Foundation](https://github.com/Strong-Foundation)
**📥 Contributions welcome** | **💬 Issues encouraged** | **⭐ Star this repo to show support**
