#!/usr/bin/env python3
"""
Create crisp menubar icons using the simple tree design.
"""

from PIL import Image, ImageDraw
import os


def create_simple_tree(size, color, padding_ratio=0.12):
    """
    Simple tree: trunk + circular crown. Clean at small sizes.
    """
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    padding = int(size * padding_ratio)
    cx = size // 2
    base_y = size - padding
    top_y = padding

    # Trunk
    trunk_width = max(2, size * 0.12)
    trunk_height = (base_y - top_y) * 0.38
    trunk_top = base_y - trunk_height

    draw.line([(cx, base_y), (cx, trunk_top)], fill=color, width=int(trunk_width))

    # Crown - filled circle
    crown_center_y = trunk_top - (base_y - top_y) * 0.28
    crown_radius = (base_y - top_y) * 0.32

    draw.ellipse(
        [cx - crown_radius, crown_center_y - crown_radius,
         cx + crown_radius, crown_center_y + crown_radius],
        fill=color
    )

    return img


def create_supersampled(target_size, color, supersample=4):
    """Create at higher resolution and downsample for anti-aliasing."""
    large_size = target_size * supersample
    large_img = create_simple_tree(large_size, color)
    return large_img.resize((target_size, target_size), Image.Resampling.LANCZOS)


def main():
    output_dir = "/Users/iheanyi/development/grove/menubar/GroveMenubar/Sources/GroveMenubar/Resources"
    os.makedirs(output_dir, exist_ok=True)

    black = (0, 0, 0, 255)

    # Menubar sizes - create crisp versions
    sizes = {
        "MenuBarIcon": 18,
        "MenuBarIcon@2x": 36,
        "MenuBarIcon-22": 22,
        "MenuBarIcon-22@2x": 44,
    }

    print("Creating crisp menubar icons...")

    for name, size in sizes.items():
        img = create_supersampled(size, black, supersample=8)
        path = os.path.join(output_dir, f"{name}.png")
        img.save(path)
        print(f"  {name}: {size}px")

    # Preview for verification
    preview = create_supersampled(128, black, supersample=4)
    preview.save(os.path.join(output_dir, "preview.png"))
    print(f"\nPreview saved for verification")

    print("Done!")


if __name__ == "__main__":
    main()
