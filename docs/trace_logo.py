#!/usr/bin/env python3
"""
Trace the Grove logo from the reference image.
"""

from PIL import Image, ImageDraw, ImageFilter
import os

def trace_reference():
    """
    Trace the reference logo by extracting the shape directly.
    """
    ref_path = "/System/Volumes/Data/Users/iheanyi/Downloads/grove.jpg"
    ref = Image.open(ref_path).convert('RGBA')

    # The reference has two versions side by side
    # Left half is dark on light, right half is light on dark
    # Let's extract the left half (dark logo on light background)

    width, height = ref.size
    left_half = ref.crop((0, 0, width // 2, height))

    # Convert to grayscale for thresholding
    gray = left_half.convert('L')

    # Threshold to get just the logo (dark pixels)
    threshold = 128
    binary = gray.point(lambda p: 255 if p < threshold else 0)

    # Convert back to RGBA with transparency
    # Black pixels become the logo, white becomes transparent
    result = Image.new('RGBA', binary.size, (0, 0, 0, 0))
    pixels = binary.load()
    result_pixels = result.load()

    for y in range(binary.height):
        for x in range(binary.width):
            if pixels[x, y] == 255:  # Logo pixel (was dark in original)
                result_pixels[x, y] = (0, 0, 0, 255)

    return result, binary


def clean_and_resize(img, target_size, padding=0.1):
    """
    Clean up the traced image and resize to target.
    """
    # Find bounding box of non-transparent pixels
    bbox = img.getbbox()
    if bbox:
        # Crop to content
        cropped = img.crop(bbox)

        # Calculate size with padding
        cw, ch = cropped.size
        max_dim = max(cw, ch)

        # Create square canvas with padding
        padded_size = int(max_dim / (1 - 2 * padding))
        canvas = Image.new('RGBA', (padded_size, padded_size), (0, 0, 0, 0))

        # Center the logo
        x = (padded_size - cw) // 2
        y = (padded_size - ch) // 2
        canvas.paste(cropped, (x, y), cropped)

        # Resize to target
        return canvas.resize((target_size, target_size), Image.Resampling.LANCZOS)

    return img.resize((target_size, target_size), Image.Resampling.LANCZOS)


def create_light_version(dark_img):
    """
    Create light (white) version from dark (black) version.
    """
    result = Image.new('RGBA', dark_img.size, (0, 0, 0, 0))
    pixels = dark_img.load()
    result_pixels = result.load()

    for y in range(dark_img.height):
        for x in range(dark_img.width):
            r, g, b, a = pixels[x, y]
            if a > 0:  # Has content
                result_pixels[x, y] = (255, 255, 255, a)

    return result


def main():
    output_dir = "/Users/iheanyi/development/grove/docs/logo"
    os.makedirs(output_dir, exist_ok=True)

    print("Tracing reference logo...")

    # Trace the reference
    traced, binary = trace_reference()

    # Save raw trace for inspection
    traced.save(os.path.join(output_dir, "traced-raw.png"))
    print("  Saved raw trace")

    # Clean and create different sizes
    sizes = [64, 128, 256, 512, 1024]

    for size in sizes:
        # Dark version
        dark = clean_and_resize(traced, size, padding=0.08)
        dark_path = os.path.join(output_dir, f"grove-traced-{size}-dark.png")
        dark.save(dark_path)

        # Light version
        light = create_light_version(dark)
        light_path = os.path.join(output_dir, f"grove-traced-{size}-light.png")
        light.save(light_path)

        print(f"  {size}px created")

    # Create comparison sheet
    print("\nCreating comparison...")

    light_bg = (245, 245, 245, 255)
    dark_bg = (40, 40, 40, 255)

    sheet_w, sheet_h = 800, 450
    sheet = Image.new('RGBA', (sheet_w, sheet_h), light_bg)
    draw = ImageDraw.Draw(sheet)

    # Dark bg on right
    draw.rectangle([sheet_w//2, 0, sheet_w, sheet_h], fill=dark_bg)

    preview_size = 350
    y = (sheet_h - preview_size) // 2

    dark_preview = clean_and_resize(traced, preview_size, padding=0.08)
    light_preview = create_light_version(dark_preview)

    x_left = (sheet_w // 4) - (preview_size // 2)
    x_right = (sheet_w * 3 // 4) - (preview_size // 2)

    sheet.paste(dark_preview, (x_left, y), dark_preview)
    sheet.paste(light_preview, (x_right, y), light_preview)

    comparison_path = os.path.join(output_dir, "grove-traced-comparison.png")
    sheet.save(comparison_path)
    print(f"Comparison: {comparison_path}")

    # Create macOS iconset
    print("\nCreating macOS iconset...")
    iconset_dir = os.path.join(output_dir, "AppIcon.iconset")
    os.makedirs(iconset_dir, exist_ok=True)

    icon_sizes = [16, 32, 64, 128, 256, 512]
    for s in icon_sizes:
        dark = clean_and_resize(traced, s, padding=0.08)
        dark.save(os.path.join(iconset_dir, f"icon_{s}x{s}.png"))
        if s <= 512:
            dark_2x = clean_and_resize(traced, s * 2, padding=0.08)
            dark_2x.save(os.path.join(iconset_dir, f"icon_{s}x{s}@2x.png"))

    print(f"Iconset: {iconset_dir}")
    print("\nTo create .icns:")
    print(f"  iconutil -c icns {iconset_dir}")

    print("\nDone!")


if __name__ == "__main__":
    main()
