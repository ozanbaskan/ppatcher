import sys

import os
import hashlib
import requests
import subprocess
import threading
import queue
import tkinter as tk
from tkinter import ttk, messagebox
import ctypes

from config import MANIFEST_URL, EXECUTABLE, TITLE, GEOMETRY

CURRENT_DIRECTORY = os.getcwd()

progress_queue = queue.Queue()


class UpdateThread(threading.Thread):
    def __init__(self):
        super().__init__()
        self.stop_event = threading.Event()
        self.last_progress = 0
        self.total_size = 0
        self.downloaded_size = 0
        self.lock = threading.Lock()

    def _update_progress(self, increment=0):
        if self.total_size > 0:
            new_progress = int(
                ((self.downloaded_size + increment) / self.total_size) * 100
            )
            if new_progress - self.last_progress >= 2 or new_progress == 100:
                self.last_progress = new_progress
                progress_queue.put(("progress", new_progress))

    def run(self):
        try:
            response = requests.get(MANIFEST_URL, timeout=10)
            manifest = response.json()

            updates_needed = []
            self.total_size = 0
            file_count = len(manifest["files"])

            progress_queue.put(("mode", "check"))
            for i, (rel_path, fileinfo) in enumerate(manifest["files"].items(), 1):
                if self.stop_event.is_set():
                    return

                local_path = os.path.join(CURRENT_DIRECTORY, *rel_path.split("/"))
                if get_file_hash(local_path) != fileinfo["hash"]:
                    updates_needed.append((rel_path, fileinfo))
                    self.total_size += fileinfo.get("size", 0)

                progress = int((i / file_count) * 100)
                if progress - self.last_progress >= 2:
                    self.last_progress = progress
                    progress_queue.put(("progress", progress))

            if updates_needed:
                progress_queue.put(("mode", "download"))
                self.downloaded_size = 0
                self.last_progress = 0

                for rel_path, fileinfo in updates_needed:
                    if self.stop_event.is_set():
                        return

                    progress_queue.put(("status", f"Updating {rel_path}..."))
                    self.download_file(fileinfo["url"], rel_path)

            progress_queue.put(("done", None))

        except Exception as e:
            progress_queue.put(("error", str(e)))

    def download_file(self, url, rel_path):
        local_path = os.path.join(CURRENT_DIRECTORY, *rel_path.split("/"))
        os.makedirs(os.path.dirname(local_path), exist_ok=True)

        try:
            with requests.get(url, stream=True, timeout=10) as response:
                response.raise_for_status()

                with open(local_path, "wb") as f:
                    for chunk in response.iter_content(131072):  # 128KB chunks
                        if self.stop_event.is_set():
                            return

                        if chunk:
                            f.write(chunk)
                            self.downloaded_size += len(chunk)
                            self._update_progress()

                return True
        except Exception as e:
            print(f"Failed to download {url}: {str(e)}")
            return False


def get_file_hash(filepath):
    if not os.path.exists(filepath):
        return None
    hasher = hashlib.md5()
    with open(filepath, "rb") as file:
        while chunk := file.read(1048576):
            hasher.update(chunk)
    return hasher.hexdigest()


class Application(tk.Tk):
    def __init__(self):
        super().__init__()
        self.current_mode = "check"
        self.withdraw()

        style = ttk.Style()
        style.theme_use("clam")
        style.configure("Horizontal.TProgressbar", troughcolor="#444", thickness=50)

        self.title(TITLE)
        self.geometry(GEOMETRY)

        self.after(50, self.start_update_check)
        self.update_thread = None

        self.status_label = tk.Label(
            self, text="Checking for updates...", font=("Arial", 12)
        )
        self.status_label.pack(pady=10)

        self.progress = ttk.Progressbar(self, length=300, mode="determinate")
        self.progress.pack(pady=10)

        self.update_button = tk.Button(
            self, text="Check for Updates", command=self.start_update_check
        )
        self.update_button.pack(pady=5)

        self.launch_button = tk.Button(self, text="Launch", command=self.launch)
        self.launch_button.pack(pady=5)

        self.deiconify()
        self.check_queue()
        self.after(100, self.start_update_check)

    def start_update_check(self):
        self.update_button.config(state=tk.DISABLED)
        self.launch_button.config(state=tk.DISABLED)
        self.status_label.config(text="Checking for updates...")
        self.progress["value"] = 0
        self.update_thread = UpdateThread()
        self.update_thread.start()

    def check_queue(self):
        try:
            while True:
                msg_type, data = progress_queue.get_nowait()

                if msg_type == "mode":
                    self.current_mode = data
                    self.progress["value"] = 0
                    self.last_progress = 0

                elif msg_type == "progress":
                    self.progress["value"] = data

                elif msg_type == "status":
                    self.status_label.config(text=data)

                elif msg_type == "error":
                    messagebox.showerror("Error", data)
                    self.enable_buttons()
                    self.status_label.config(text="Update failed")

                elif msg_type == "done":
                    self.enable_buttons()
                    self.progress["value"] = 0
                    self.last_progress = 0
                    self.status_label.config(text="Ready")

        except queue.Empty:
            pass
        self.after(50, self.check_queue)

    def enable_buttons(self):
        self.update_button.config(state=tk.NORMAL)
        self.launch_button.config(state=tk.NORMAL)

    def launch(self):
        exe_path = os.path.abspath(os.path.join(CURRENT_DIRECTORY, EXECUTABLE))
        print(exe_path)
        if sys.platform == "win32":
            try:
                exe_path = exe_path.replace("/", "\\")
                result = ctypes.windll.shell32.ShellExecuteW(
                    None,  # hwnd
                    "runas",  # Operation (admin request)
                    exe_path,  # Application path
                    None,  # Parameters
                    None,  # Directory
                    1,  # SW_SHOWNORMAL
                )

                if result <= 32:
                    error_messages = {
                        2: "File not found",
                        3: "Path not found",
                        5: "Access denied - User canceled UAC prompt",
                        740: "Requires elevation",
                    }
                    error_msg = error_messages.get(result, f"Error code: {result}")

                    messagebox.showerror("Launch Error", error_msg)
                else:
                    self.status_label.config(text="Launched!")

            except Exception as e:
                messagebox.showerror("Error", f"Failed to launch: {str(e)}")

        else:
            try:
                os.chmod(exe_path, 0o755)
                subprocess.Popen([exe_path], start_new_session=True)
            except Exception as e:
                messagebox.showerror("Error", f"Launch failed: {str(e)}")

    def on_closing(self):
        if self.update_thread and self.update_thread.is_alive():
            self.update_thread.stop_event.set()
            self.update_thread.join()
        self.destroy()


if __name__ == "__main__":
    app = Application()
    app.protocol("WM_DELETE_WINDOW", app.on_closing)
    app.mainloop()
