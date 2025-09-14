import os

def tree(path, prefix=""):
    exclude = {"node_modules", ".git"}  # folders to skip
    entries = [e for e in os.listdir(path) if e not in exclude]
    entries.sort()
    for i, entry in enumerate(entries):
        full = os.path.join(path, entry)
        connector = "└── " if i == len(entries) - 1 else "├── "
        print(prefix + connector + entry)
        if os.path.isdir(full):
            extension = "    " if i == len(entries) - 1 else "│   "
            tree(full, prefix + extension)

tree(r"G:\go_internist")
