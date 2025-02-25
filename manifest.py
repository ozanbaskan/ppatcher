import os
import hashlib
import json
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

from config import FOLDER_URL, FILES_VERSION, MANIFEST_DIRECTORY, FILES_ROOT_DIRECTORY

BASE_URL = FOLDER_URL.rstrip("/") + "/"
BUFFER_SIZE = 128 * 1024


def get_file_hash_size(file_path):
    """Calculate MD5 hash with optimized buffer size."""
    md5 = hashlib.md5()
    total_size = 0
    with open(file_path, "rb") as file:
        while chunk := file.read(BUFFER_SIZE):
            total_size += len(chunk)
            md5.update(chunk)
    return md5.hexdigest(), total_size


def process_file(file_entry):
    rel_path, abs_path = file_entry
    hash, size = get_file_hash_size(abs_path)
    return (rel_path, {"hash": hash, "size": size, "url": f"{BASE_URL}{rel_path}"})


def generate_manifest():
    manifest = {"version": FILES_VERSION, "files": {}}
    file_entries = []
    root_path = Path(FILES_ROOT_DIRECTORY)

    for file_path in root_path.rglob("*"):
        if file_path.is_file():
            rel_path = str(file_path.relative_to(root_path)).replace("\\", "/")
            file_entries.append((rel_path, str(file_path)))

    with ThreadPoolExecutor() as executor:
        results = executor.map(process_file, file_entries)

        for rel_path, file_data in results:
            manifest["files"][rel_path] = file_data

    with open(MANIFEST_DIRECTORY, "w") as f:
        json.dump(manifest, f, indent=4)

    print(f"Manifest generated successfully with {len(file_entries)} files")


if __name__ == "__main__":
    print("Starting manifest generation...")
    generate_manifest()
