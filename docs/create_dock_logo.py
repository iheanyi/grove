#!/usr/bin/env python3
"""
Grove Dock Logo Generator

Recreates the branching tree with git icon design for macOS dock.
"""

from PIL import Image, ImageDraw
import math
import os


def draw_branch(draw, start, angle, length, width, depth, max_depth, color):
    """
    Recursively draw branches of the tree.
    """
    if depth > max_depth or length < 2:
        return

    # Calculate end point
    end_x = start[0] + length * math.sin(math.radians(angle))
    end_y = start[1] - length * math.cos(math.radians(angle))
    end = (end_x, end_y)

    # Draw this branch
    draw.line([start, end], fill=color, width=max(1, int(width)))

    # Draw rounded cap at branch tip
    cap_radius = width / 2
    draw.ellipse(
        [end_x - cap_radius, end_y - cap_radius,
         end_x + cap_radius, end_y + cap_radius],
        fill=color
    )

    # Recursively draw sub-branches
    # Reduce length and width for child branches
    new_length = length * 0.65
    new_width = width * 0.7

    # Branch angles - spread out from current direction
    spread = 35 + depth * 5  # Wider spread at higher depths

    # Left branch
    draw_branch(draw, end, angle - spread, new_length, new_width, depth + 1, max_depth, color)
    # Right branch
    draw_branch(draw, end, angle + spread, new_length, new_width, depth + 1, max_depth, color)


def draw_git_tag(draw, center_x, center_y, size, color, bg_color=None):
    """
    Draw the git branch/tag icon (tilted rounded rectangle with branch symbol).
    """
    # The tag is a tilted rounded square
    angle = 45  # degrees
    half_size = size / 2

    # Corner points of rotated square
    corners = []
    for i in range(4):
        corner_angle = math.radians(angle + i * 90)
        x = center_x + half_size * math.cos(corner_angle)
        y = center_y + half_size * math.sin(corner_angle)
        corners.append((x, y))

    # Draw the tag shape
    draw.polygon(corners, fill=color)

    # Draw the hole/cutout for the git symbol (if bg_color provided)
    if bg_color:
        # Inner rounded rectangle (smaller, creates border effect)
        inner_size = size * 0.7
        inner_half = inner_size / 2
        inner_corners = []
        for i in range(4):
            corner_angle = math.radians(angle + i * 90)
            x = center_x + inner_half * math.cos(corner_angle)
            y = center_y + inner_half * math.sin(corner_angle)
            inner_corners.append((x, y))
        draw.polygon(inner_corners, fill=bg_color)

        # Git branch symbol inside
        # Main line (vertical in the tag's coordinate space, so diagonal in screen space)
        line_length = inner_size * 0.5
        line_start = (center_x - line_length * 0.4, center_y + line_length * 0.4)
        line_end = (center_x + line_length * 0.4, center_y - line_length * 0.4)
        branch_width = max(2, size * 0.08)
        draw.line([line_start, line_end], fill=color, width=int(branch_width))

        # Branch coming off
        branch_start = (center_x, center_y)
        branch_end = (center_x + line_length * 0.35, center_y + line_length * 0.1)
        draw.line([branch_start, branch_end], fill=color, width=int(branch_width))

        # Dots at branch points
        dot_radius = size * 0.06
        for point in [line_start, line_end, branch_end]:
            draw.ellipse(
                [point[0] - dot_radius, point[1] - dot_radius,
                 point[0] + dot_radius, point[1] + dot_radius],
                fill=color
            )


def create_grove_dock_icon(size, color, bg_color=None, padding_ratio=0.1):
    """
    Create the full grove dock icon with branching tree and git tag.
    """
    if bg_color is None:
        img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    else:
        img = Image.new('RGBA', (size, size), bg_color)

    draw = ImageDraw.Draw(img)

    padding = int(size * padding_ratio)
    cx = size // 2

    # Tree parameters
    tree_top = padding
    tree_bottom = size - padding - size * 0.12  # Leave room for git tag

    trunk_length = (tree_bottom - tree_top) * 0.25
    trunk_width = size * 0.06

    # Starting point (base of trunk, will branch upward)
    trunk_base = (cx, tree_bottom)
    trunk_top = (cx, tree_bottom - trunk_length)

    # Draw main trunk
    draw.line([trunk_base, trunk_top], fill=color, width=int(trunk_width))

    # Branch from trunk top
    branch_length = trunk_length * 1.2
    branch_width = trunk_width * 0.85

    # Draw branches recursively from trunk top
    # Two main branches going left and right
    draw_branch(draw, trunk_top, -25, branch_length, branch_width, 0, 4, color)
    draw_branch(draw, trunk_top, 25, branch_length, branch_width, 0, 4, color)

    # Git tag at the bottom
    tag_size = size * 0.18
    tag_center_y = tree_bottom + size * 0.08
    # Offset slightly to the left to match reference
    tag_center_x = cx - size * 0.08

    draw_git_tag(draw, tag_center_x, tag_center_y, tag_size, color, bg_color if bg_color else (0, 0, 0, 0))

    # Connection line from trunk to tag
    conn_width = max(2, trunk_width * 0.4)
    draw.line([(cx, tree_bottom), (tag_center_x, tag_center_y - tag_size * 0.5)],
              fill=color, width=int(conn_width))

    return img


def create_supersampled(target_size, color, bg_color=None, supersample=4):
    """Create at higher resolution and downsample for anti-aliasing."""
    large_size = target_size * supersample
    large_img = create_grove_dock_icon(large_size, color, bg_color)
    return large_img.resize((target_size, target_size), Image.Resampling.LANCZOS)


def create_macos_iconset(output_dir, color_dark, color_light):
    """
    Create all sizes needed for a macOS .icns file.
    Sizes: 16, 32, 64, 128, 256, 512, 1024 (and @2x variants)
    """
    iconset_dir = os.path.join(output_dir, "AppIcon.iconset")
    os.makedirs(iconset_dir, exist_ok=True)

    # macOS icon sizes
    sizes = [16, 32, 64, 128, 256, 512, 1024]

    for base_size in sizes:
        # 1x version
        img = create_supersampled(base_size, color_dark, supersample=4)
        img.save(os.path.join(iconset_dir, f"icon_{base_size}x{base_size}.png"))

        # 2x version (for retina)
        if base_size <= 512:
            img_2x = create_supersampled(base_size * 2, color_dark, supersample=4)
            img_2x.save(os.path.join(iconset_dir, f"icon_{base_size}x{base_size}@2x.png"))

    print(f"Created iconset at: {iconset_dir}")
    print("To create .icns file, run:")
    print(f"  iconutil -c icns {iconset_dir}")

    return iconset_dir


def main():
    output_dir = "/Users/iheanyi/development/grove/docs/logo"
    os.makedirs(output_dir, exist_ok=True)

    # Colors
    black = (0, 0, 0, 255)
    white = (255, 255, 255, 255)
    light_bg = (245, 245, 245, 255)
    dark_bg = (35, 35, 35, 255)

    print("Generating Grove dock icon...")
    print("=" * 50)

    # Create preview sizes
    sizes = [64, 128, 256, 512, 1024]

    for size in sizes:
        # Dark version (for light backgrounds)
        img_dark = create_supersampled(size, black, supersample=4)
        dark_path = os.path.join(output_dir, f"grove-dock-{size}-dark.png")
        img_dark.save(dark_path)

        # Light version (for dark backgrounds)
        img_light = create_supersampled(size, white, supersample=4)
        light_path = os.path.join(output_dir, f"grove-dock-{size}-light.png")
        img_light.save(light_path)

        print(f"  {size}px: {dark_path}")

    # Create comparison sheet
    print("\nCreating comparison sheet...")
    sheet_size = 800
    sheet = Image.new('RGBA', (sheet_size, sheet_size), light_bg)
    draw = ImageDraw.Draw(sheet)

    # Dark background for bottom half
    draw.rectangle([0, sheet_size//2, sheet_size, sheet_size], fill=dark_bg)

    # Place logos
    preview_size = 300
    y_light = (sheet_size // 4) - (preview_size // 2)
    y_dark = (sheet_size * 3 // 4) - (preview_size // 2)
    x = (sheet_size - preview_size) // 2

    img_dark = create_supersampled(preview_size, black, supersample=4)
    img_light = create_supersampled(preview_size, white, supersample=4)

    sheet.paste(img_dark, (x, y_light), img_dark)
    sheet.paste(img_light, (x, y_dark), img_light)

    comparison_path = os.path.join(output_dir, "grove-dock-comparison.png")
    sheet.save(comparison_path)
    print(f"Comparison: {comparison_path}")

    # Create macOS iconset
    print("\nCreating macOS iconset...")
    create_macos_iconset(output_dir, black, white)

    print("\n" + "=" * 50)
    print("Dock icon generation complete!")


if __name__ == "__main__":
    main()
