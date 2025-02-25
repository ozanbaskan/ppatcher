import os
import hashlib
import json

from config import FILES_DIRECTORY, FOLDER_URL, FILES_VERSION, MANIFEST_DIRECTORY

def get_file_hash(file_path):
    hash_sha256 = hashlib.sha256()
    with open(file_path, "rb") as file:
        for chunk in iter(lambda: file.read(4096), b""):
            hash_sha256.update(chunk)
    return hash_sha256.hexdigest()

def generate_manifest(directory):
    manifest = {"files": {}}
    
    for root, dirs, files in os.walk(directory):
        for file in files:
            file_path = os.path.join(root, file)
            file_hash = get_file_hash(file_path)
            
            manifest["version"] = FILES_VERSION

            manifest["files"][file] = {
                "hash": file_hash,
                "url": f"{FOLDER_URL}/{file}" 
            }
    
    with open(MANIFEST_DIRECTORY, "w") as f:
        json.dump(manifest, f, indent=4)

    print("Manifest generated successfully!")

if __name__ == "__main__":
    generate_manifest(FILES_DIRECTORY)
