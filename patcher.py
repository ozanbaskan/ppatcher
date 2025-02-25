import os
import sys
import hashlib
import requests
import subprocess
import tkinter as tk
from tkinter import ttk, messagebox

from config import MANIFEST_URL, EXECUTABLE, TITLE, GEOMETRY

CURRENT_DIRECTORY = os.getcwd()

root = tk.Tk()
root.title(TITLE)
root.geometry(GEOMETRY)

status_label = tk.Label(root, text="Checking for updates...", font=("Arial", 12))
status_label.pack(pady=10)

progress = ttk.Progressbar(root, length=300, mode="determinate")
progress.pack(pady=10)

update_button = tk.Button(root, text="Check for Updates", state=tk.DISABLED)
update_button.pack(pady=5)

launch_button = tk.Button(root, text="Launch", state=tk.DISABLED)
launch_button.pack(pady=5)

def download_file(url, local_path):
    response = requests.get(url, stream=True)
    total_size = int(response.headers.get("content-length", 0))
    progress["maximum"] = total_size

    with open(local_path, "wb") as file:
        downloaded_size = 0
        for chunk in response.iter_content(1024):
            if chunk:
                file.write(chunk)
                downloaded_size += len(chunk)
                progress["value"] = downloaded_size
                root.update_idletasks()
    print(f"Downloaded: {local_path}")

def get_file_hash(filepath):
    if not os.path.exists(filepath):
        return None
    hasher = hashlib.sha256()
    with open(filepath, "rb") as file:
        for chunk in iter(lambda: file.read(4096), b""):
            hasher.update(chunk)
    return hasher.hexdigest()

def check_for_updates():
    update_button.config(state=tk.DISABLED)
    launch_button.config(state=tk.DISABLED)
    status_label.config(text="Checking for updates...")
    root.update_idletasks()

    try:
        response = requests.get(MANIFEST_URL)

        if response.status_code != 200:
            status_label.config(text="Failed to fetch file data.")
            return

        manifest = response.json()
        updates_found = False

        for filename, fileinfo in manifest["files"].items():
            local_path = os.path.join(CURRENT_DIRECTORY, filename)
            if get_file_hash(local_path) != fileinfo["hash"]:
                updates_found = True
                status_label.config(text=f"Updating {filename}...")
                root.update_idletasks()
                download_file(fileinfo["url"], local_path)

        if updates_found:
            status_label.config(text="Update complete!")
        else:
            status_label.config(text="No updates needed.")

    except Exception as e:
        status_label.config(text="Update check failed.")
        messagebox.showerror("Error", str(e))

    update_button.config(state=tk.NORMAL)
    launch_button.config(state=tk.NORMAL)

def launch():
    path = os.path.join(CURRENT_DIRECTORY, EXECUTABLE)
    if sys.platform != 'win32':
        os.chmod(path, 0o755)

    if os.path.exists(path):
        subprocess.Popen([path])
        status_label.config(text="Launched!")
    else:
        status_label.config(text="Not found.")

root.after(100, check_for_updates)

update_button.config(command=check_for_updates)
launch_button.config(command=launch)

root.mainloop()
