#!/usr/bin/env python3
"""
Create properly sized menubar icons from the traced logo.

macOS menubar icons should be:
- 18x18 or 22x22 for standard
- 36x36 or 44x44 for @2x retina
- Template images (black with alpha, macOS handles dark mode)
"""

from PIL import Image
import os

def create_menubar_icons():
    # Load the high-res traced logo
    source = Image.open("/Users/iheanyi/development/grove/docs/logo/grove-traced-512-dark.png")

    output_dir = "/Users/iheanyi/development/grove/menubar/GroveMenubar/Sources/GroveMenubar/Resources"
    os.makedirs(output_dir, exist_ok=True)

    # Menubar icon sizes
    sizes = {
        "MenuBarIcon": 18,
        "MenuBarIcon@2x": 36,
        "MenuBarIcon-22": 22,
        "MenuBarIcon-22@2x": 44,
    }

    for name, size in sizes.items():
        # Resize with high quality
        resized = source.resize((size, size), Image.Resampling.LANCZOS)

        # Save
        path = os.path.join(output_dir, f"{name}.png")
        resized.save(path)
        print(f"Created: {path}")

    # Also copy the dock icon (AppIcon.icns)
    import shutil
    icns_src = "/Users/iheanyi/development/grove/docs/logo/AppIcon.icns"
    icns_dst = os.path.join(output_dir, "AppIcon.icns")
    shutil.copy(icns_src, icns_dst)
    print(f"Copied: {icns_dst}")

    print("\nDone!")

if __name__ == "__main__":
    create_menubar_icons()
