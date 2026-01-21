#!/usr/bin/env python3
"""
Grove Logo Generator

Creates a minimalist grove/tree icon suitable for macOS menubar.
Design: Three ascending strokes suggesting tree branches / git worktrees.
"""

from PIL import Image, ImageDraw
import math
import os

def create_grove_logo(size, stroke_width, color, bg_color=None, padding_ratio=0.15):
    """
    Create a grove logo at the specified size.

    The design: Three stylized tree/branch forms rising from a common base,
    representing a grove of trees and the branching nature of git worktrees.
    """
    # Create image with transparency if no bg_color
    if bg_color is None:
        img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    else:
        img = Image.new('RGBA', (size, size), bg_color)

    draw = ImageDraw.Draw(img)

    # Calculate dimensions with padding
    padding = int(size * padding_ratio)
    inner_size = size - (padding * 2)

    # Center point
    cx = size // 2

    # Base Y (bottom of the icon area)
    base_y = size - padding

    # Top Y (top of the icon area)
    top_y = padding

    # Height of the main trunk before branching
    trunk_height = inner_size * 0.35
    trunk_top_y = base_y - trunk_height

    # Branch spread at top
    spread = inner_size * 0.35

    # Draw with anti-aliasing by using supersampling
    # We'll draw at 4x size then downscale

    # For cleaner lines, use a simple geometric approach
    # Three lines emanating from a trunk point

    # Main vertical trunk (center)
    trunk_bottom = (cx, base_y)
    trunk_top = (cx, trunk_top_y)

    # Three branch endpoints at top
    left_top = (cx - spread, top_y + inner_size * 0.1)
    center_top = (cx, top_y)
    right_top = (cx + spread, top_y + inner_size * 0.1)

    # Branch point (where branches split from trunk)
    branch_y = trunk_top_y + inner_size * 0.05
    branch_point = (cx, branch_y)

    # Draw trunk
    draw.line([trunk_bottom, branch_point], fill=color, width=stroke_width)

    # Draw three branches with slight curves implied by the angles
    # Left branch
    draw.line([branch_point, left_top], fill=color, width=stroke_width)

    # Center branch (continues straight up)
    draw.line([branch_point, center_top], fill=color, width=stroke_width)

    # Right branch
    draw.line([branch_point, right_top], fill=color, width=stroke_width)

    # Add small circular terminals at branch tips for polish
    terminal_radius = max(1, stroke_width // 2)

    for point in [left_top, center_top, right_top]:
        x, y = point
        draw.ellipse(
            [x - terminal_radius, y - terminal_radius,
             x + terminal_radius, y + terminal_radius],
            fill=color
        )

    return img


def create_grove_logo_v2(size, stroke_width, color, bg_color=None, padding_ratio=0.12):
    """
    Alternative design: Abstract tree silhouette - single trunk with crown.
    More organic, suggests a tree canopy.
    """
    if bg_color is None:
        img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    else:
        img = Image.new('RGBA', (size, size), bg_color)

    draw = ImageDraw.Draw(img)

    padding = int(size * padding_ratio)

    cx = size // 2
    base_y = size - padding
    top_y = padding

    # Trunk
    trunk_width = stroke_width
    trunk_height = (base_y - top_y) * 0.4
    trunk_top = base_y - trunk_height

    draw.line([(cx, base_y), (cx, trunk_top)], fill=color, width=trunk_width)

    # Crown - three overlapping circles suggesting foliage
    crown_center_y = trunk_top - (base_y - top_y) * 0.25
    crown_radius = (base_y - top_y) * 0.28

    # Draw crown as filled circle (simple, bold)
    draw.ellipse(
        [cx - crown_radius, crown_center_y - crown_radius,
         cx + crown_radius, crown_center_y + crown_radius],
        fill=color
    )

    return img


def create_grove_logo_v3(size, stroke_width, color, bg_color=None, padding_ratio=0.1):
    """
    Design v3: Three parallel ascending lines with slight convergence at bottom.
    Minimal, geometric, suggests multiple trees in a grove.
    """
    if bg_color is None:
        img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    else:
        img = Image.new('RGBA', (size, size), bg_color)

    draw = ImageDraw.Draw(img)

    padding = int(size * padding_ratio)

    cx = size // 2
    base_y = size - padding
    top_y = padding

    # Three lines, slightly converging at bottom
    # Top spread
    top_spread = size * 0.32
    # Bottom spread (narrower - convergence)
    bottom_spread = size * 0.22

    # Heights - center tallest, sides slightly shorter
    center_height = base_y - top_y
    side_height = center_height * 0.85

    # Left line
    left_bottom = (cx - bottom_spread, base_y)
    left_top = (cx - top_spread, base_y - side_height)

    # Center line
    center_bottom = (cx, base_y)
    center_top = (cx, top_y)

    # Right line
    right_bottom = (cx + bottom_spread, base_y)
    right_top = (cx + top_spread, base_y - side_height)

    # Draw lines
    draw.line([left_bottom, left_top], fill=color, width=stroke_width)
    draw.line([center_bottom, center_top], fill=color, width=stroke_width)
    draw.line([right_bottom, right_top], fill=color, width=stroke_width)

    # Rounded terminals
    terminal_radius = max(1, stroke_width * 0.6)

    for point in [left_top, center_top, right_top]:
        x, y = point
        draw.ellipse(
            [x - terminal_radius, y - terminal_radius,
             x + terminal_radius, y + terminal_radius],
            fill=color
        )

    return img


def create_grove_logo_v4(size, stroke_width, color, bg_color=None, padding_ratio=0.12):
    """
    Design v4: Stylized 'G' that incorporates a tree branch.
    Letter-based but organic.
    """
    if bg_color is None:
        img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    else:
        img = Image.new('RGBA', (size, size), bg_color)

    draw = ImageDraw.Draw(img)

    padding = int(size * padding_ratio)
    inner = size - padding * 2

    cx, cy = size // 2, size // 2
    radius = inner // 2

    # Draw a G shape using arc
    # G is essentially a C with a horizontal bar
    bbox = [padding, padding, size - padding, size - padding]

    # Draw arc (open circle) - from about 45 degrees to 315 degrees
    # PIL angles: 0 is 3 o'clock, goes counter-clockwise
    draw.arc(bbox, start=45, end=315, fill=color, width=stroke_width)

    # Horizontal bar of G
    bar_y = cy
    bar_left = cx
    bar_right = cx + radius * 0.5
    draw.line([(bar_left, bar_y), (bar_right, bar_y)], fill=color, width=stroke_width)

    # Small branch/leaf accent on top right
    branch_start = (cx + radius * 0.5, padding + stroke_width)
    branch_end = (cx + radius * 0.8, padding - stroke_width * 2)
    draw.line([branch_start, branch_end], fill=color, width=max(1, stroke_width // 2))

    return img


def create_supersampled(create_func, target_size, supersample=4, **kwargs):
    """Create logo at higher resolution and downsample for anti-aliasing."""
    large_size = target_size * supersample
    large_stroke = kwargs.get('stroke_width', 2) * supersample

    # Create at large size
    large_kwargs = kwargs.copy()
    large_kwargs['stroke_width'] = large_stroke
    large_img = create_func(large_size, **large_kwargs)

    # Downsample with high-quality resampling
    return large_img.resize((target_size, target_size), Image.Resampling.LANCZOS)


def main():
    output_dir = "/Users/iheanyi/development/grove/docs/logo"
    os.makedirs(output_dir, exist_ok=True)

    # Colors
    black = (0, 0, 0, 255)
    white = (255, 255, 255, 255)

    # Sizes for different uses
    sizes = {
        'menubar': 22,
        'menubar@2x': 44,
        'small': 64,
        'medium': 128,
        'large': 256,
        'xlarge': 512,
    }

    # Design functions to try
    designs = {
        'branching': create_grove_logo,      # Three branches from trunk
        'tree': create_grove_logo_v2,         # Simple tree silhouette
        'grove': create_grove_logo_v3,        # Three parallel lines (grove of trees)
    }

    print("Generating Grove logo variants...")
    print("=" * 50)

    for design_name, design_func in designs.items():
        print(f"\nDesign: {design_name}")

        for size_name, size in sizes.items():
            # Calculate appropriate stroke width for size
            if size <= 22:
                stroke = 2
            elif size <= 44:
                stroke = 3
            elif size <= 64:
                stroke = 4
            elif size <= 128:
                stroke = 6
            elif size <= 256:
                stroke = 10
            else:
                stroke = 16

            # Create dark version (for light backgrounds)
            img_dark = create_supersampled(
                design_func, size,
                supersample=4,
                stroke_width=stroke,
                color=black
            )

            # Create light version (for dark backgrounds)
            img_light = create_supersampled(
                design_func, size,
                supersample=4,
                stroke_width=stroke,
                color=white
            )

            # Save
            dark_path = os.path.join(output_dir, f"grove-{design_name}-{size_name}-dark.png")
            light_path = os.path.join(output_dir, f"grove-{design_name}-{size_name}-light.png")

            img_dark.save(dark_path)
            img_light.save(light_path)

            print(f"  {size_name} ({size}px): {dark_path}")

    # Create a comparison sheet
    print("\nCreating comparison sheet...")

    sheet_size = 800
    sheet = Image.new('RGBA', (sheet_size, sheet_size), (240, 240, 240, 255))
    draw = ImageDraw.Draw(sheet)

    # Draw dark background for bottom half
    draw.rectangle([0, sheet_size//2, sheet_size, sheet_size], fill=(30, 30, 30, 255))

    # Place logos
    preview_size = 128
    y_light = 100
    y_dark = sheet_size // 2 + 100

    for i, (design_name, design_func) in enumerate(designs.items()):
        x = 100 + i * 220

        # Create preview versions
        img_dark = create_supersampled(
            design_func, preview_size,
            supersample=4,
            stroke_width=8,
            color=black
        )
        img_light = create_supersampled(
            design_func, preview_size,
            supersample=4,
            stroke_width=8,
            color=white
        )

        # Paste on light background
        sheet.paste(img_dark, (x, y_light), img_dark)

        # Paste on dark background
        sheet.paste(img_light, (x, y_dark), img_light)

    # Add labels (simple text positioning)
    comparison_path = os.path.join(output_dir, "grove-logo-comparison.png")
    sheet.save(comparison_path)
    print(f"Comparison sheet: {comparison_path}")

    print("\n" + "=" * 50)
    print("Logo generation complete!")
    print(f"Files saved to: {output_dir}")


if __name__ == "__main__":
    main()
