import os
from PIL import Image, ImageDraw, ImageFont

def build_tree(root, prefix=""):
    """Return a list of lines representing the tree under root."""
    entries = [e for e in os.listdir(root)
               if e not in ('node_modules', '.git')
               and not e.startswith('.git')
               and e != 'tee.py']  # skip tee.py
    entries.sort(key=lambda e: (not os.path.isdir(os.path.join(root, e)), e.lower()))

    lines = []
    for i, entry in enumerate(entries):
        path = os.path.join(root, entry)
        connector = "└── " if i == len(entries) - 1 else "├── "
        lines.append(prefix + connector + entry)
        if os.path.isdir(path):
            extension = "    " if i == len(entries) - 1 else "│   "
            lines.extend(build_tree(path, prefix + extension))
    return lines

def tree_to_jpg(root_directory, output_file="tree.jpg"):
    root_name = os.path.basename(root_directory.rstrip(os.sep))
    lines = [root_name] + build_tree(root_directory)

    # Font settings
    try:
        font = ImageFont.truetype("consola.ttf", 14)  # Windows console font
    except:
        font = ImageFont.load_default()

    # Determine image size
    max_width = max(font.getlength(line) for line in lines)
    line_height = font.getbbox("Ay")[3] + 4
    img_height = line_height * len(lines) + 10
    img_width = int(max_width) + 20

    # Create image
    img = Image.new("RGB", (img_width, img_height), color=(255, 255, 255))
    draw = ImageDraw.Draw(img)

    # Draw text
    y = 5
    for line in lines:
        draw.text((10, y), line, fill=(0, 0, 0), font=font)
        y += line_height

    img.save(output_file)
    print(f"Saved tree to {output_file}")

if __name__ == "__main__":
    root_directory = r"G:\go_internist"
    tree_to_jpg(root_directory, "tree_output.jpg")
