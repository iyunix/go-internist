import os

# directories to skip
SKIP = {'node_modules', '.git', '__pycache__'}

def print_tree(start_path='.', prefix=''):
    # separate directories and files
    try:
        entries = sorted(os.listdir(start_path))
    except PermissionError:
        return

    dirs = [d for d in entries if os.path.isdir(os.path.join(start_path, d)) and d not in SKIP]
    files = [f for f in entries if os.path.isfile(os.path.join(start_path, f))]

    total = len(dirs) + len(files)
    for index, name in enumerate(dirs + files):
        path = os.path.join(start_path, name)
        connector = "└── " if index == total - 1 else "├── "

        if os.path.isdir(path):
            print(f"{prefix}{connector}📁 {name}")
            # extend prefix for children
            new_prefix = prefix + ("    " if index == total - 1 else "│   ")
            print_tree(path, new_prefix)
        else:
            print(f"{prefix}{connector}📄 {name}")

if __name__ == '__main__':
    print("📁 Project Tree\n")
    print_tree('.')
