#!/usr/bin/env python3
"""
Grove Dock Logo Generator v2

More organic branching pattern matching the reference design.
"""

from PIL import Image, ImageDraw
import math
import os


def draw_organic_branch(draw, start, angle, length, width, depth, max_depth, color, branch_data):
    """
    Draw branches with more organic, asymmetric pattern like the reference.
    """
    if depth > max_depth or length < 3 or width < 1:
        return

    # Calculate end point
    end_x = start[0] + length * math.sin(math.radians(angle))
    end_y = start[1] - length * math.cos(math.radians(angle))
    end = (end_x, end_y)

    # Draw this branch segment
    draw.line([start, end], fill=color, width=max(1, int(width)))

    # Rounded cap at end
    cap_r = width / 2.2
    draw.ellipse([end_x - cap_r, end_y - cap_r, end_x + cap_r, end_y + cap_r], fill=color)

    # Get branching parameters for this depth
    params = branch_data.get(depth, {'spread': 40, 'length_factor': 0.6, 'width_factor': 0.7})

    new_length = length * params['length_factor']
    new_width = width * params['width_factor']
    spread = params['spread']

    # Asymmetric branching - vary the angles slightly
    left_spread = spread + (depth * 3)
    right_spread = spread + (depth * 2)

    # Left sub-branch
    draw_organic_branch(draw, end, angle - left_spread, new_length, new_width,
                       depth + 1, max_depth, color, branch_data)
    # Right sub-branch
    draw_organic_branch(draw, end, angle + right_spread, new_length, new_width,
                       depth + 1, max_depth, color, branch_data)


def draw_git_tag_diamond(draw, cx, cy, size, color, inner_color):
    """
    Draw git tag as tilted diamond with git branch symbol inside.
    """
    # Diamond points (rotated square)
    half = size / 2
    points = [
        (cx, cy - half),      # top
        (cx + half, cy),      # right
        (cx, cy + half),      # bottom
        (cx - half, cy),      # left
    ]

    # Draw outer diamond
    draw.polygon(points, fill=color)

    # Draw inner diamond (creates border effect)
    inner_half = half * 0.72
    inner_points = [
        (cx, cy - inner_half),
        (cx + inner_half, cy),
        (cx, cy + inner_half),
        (cx - inner_half, cy),
    ]
    draw.polygon(inner_points, fill=inner_color)

    # Git branch symbol inside
    # A simple branching line pattern
    line_w = max(2, size * 0.09)
    dot_r = size * 0.07

    # Main diagonal line
    main_len = inner_half * 0.6
    p1 = (cx - main_len * 0.5, cy + main_len * 0.5)
    p2 = (cx + main_len * 0.5, cy - main_len * 0.5)
    draw.line([p1, p2], fill=color, width=int(line_w))

    # Branch line
    branch_start = (cx - main_len * 0.1, cy + main_len * 0.1)
    branch_end = (cx + main_len * 0.4, cy + main_len * 0.25)
    draw.line([branch_start, branch_end], fill=color, width=int(line_w))

    # Dots at endpoints
    for p in [p1, p2, branch_end]:
        draw.ellipse([p[0]-dot_r, p[1]-dot_r, p[0]+dot_r, p[1]+dot_r], fill=color)


def create_grove_dock_icon_v2(size, color, bg_color=None, padding_ratio=0.08):
    """
    Create grove dock icon with organic branching tree and git tag.
    """
    transparent = (0, 0, 0, 0)
    if bg_color is None:
        img = Image.new('RGBA', (size, size), transparent)
        inner_color = transparent
    else:
        img = Image.new('RGBA', (size, size), bg_color)
        inner_color = bg_color

    draw = ImageDraw.Draw(img)

    padding = int(size * padding_ratio)
    cx = size // 2

    # Layout
    tree_top = padding
    tag_size = size * 0.14
    tag_bottom = size - padding
    tag_center_y = tag_bottom - tag_size * 0.6

    # Trunk starts above the tag
    trunk_bottom_y = tag_center_y - tag_size * 0.6
    trunk_top_y = trunk_bottom_y - size * 0.22

    trunk_width = size * 0.055

    # Draw trunk
    draw.line([(cx, trunk_bottom_y), (cx, trunk_top_y)], fill=color, width=int(trunk_width))

    # The trunk splits into two main curved branches
    # We'll draw them as angled lines that then branch further

    # Branch parameters by depth for organic look
    branch_data = {
        0: {'spread': 28, 'length_factor': 0.72, 'width_factor': 0.78},
        1: {'spread': 35, 'length_factor': 0.68, 'width_factor': 0.72},
        2: {'spread': 42, 'length_factor': 0.62, 'width_factor': 0.65},
        3: {'spread': 48, 'length_factor': 0.55, 'width_factor': 0.58},
        4: {'spread': 55, 'length_factor': 0.5, 'width_factor': 0.5},
    }

    # Main branch length and width
    main_branch_length = size * 0.18
    main_branch_width = trunk_width * 0.9

    # Two main branches from trunk top
    trunk_top = (cx, trunk_top_y)

    # Left main branch
    draw_organic_branch(draw, trunk_top, -22, main_branch_length, main_branch_width,
                       0, 4, color, branch_data)
    # Right main branch
    draw_organic_branch(draw, trunk_top, 22, main_branch_length, main_branch_width,
                       0, 4, color, branch_data)

    # Connection line from trunk to tag (slightly offset to left like reference)
    tag_cx = cx - size * 0.06
    conn_top = (cx, trunk_bottom_y)
    conn_bottom = (tag_cx, tag_center_y - tag_size * 0.5)
    conn_width = max(2, trunk_width * 0.5)
    draw.line([conn_top, conn_bottom], fill=color, width=int(conn_width))

    # Git tag
    draw_git_tag_diamond(draw, tag_cx, tag_center_y, tag_size, color, inner_color)

    return img


def create_supersampled_v2(target_size, color, bg_color=None, supersample=4):
    """Create at higher resolution and downsample for anti-aliasing."""
    large_size = target_size * supersample
    large_img = create_grove_dock_icon_v2(large_size, color, bg_color)
    return large_img.resize((target_size, target_size), Image.Resampling.LANCZOS)


def main():
    output_dir = "/Users/iheanyi/development/grove/docs/logo"
    os.makedirs(output_dir, exist_ok=True)

    black = (0, 0, 0, 255)
    white = (255, 255, 255, 255)
    light_bg = (245, 245, 245, 255)
    dark_bg = (40, 40, 40, 255)

    print("Generating Grove dock icon v2...")
    print("=" * 50)

    # Create various sizes
    sizes = [64, 128, 256, 512, 1024]

    for size in sizes:
        img_dark = create_supersampled_v2(size, black, supersample=4)
        dark_path = os.path.join(output_dir, f"grove-dock-v2-{size}-dark.png")
        img_dark.save(dark_path)

        img_light = create_supersampled_v2(size, white, supersample=4)
        light_path = os.path.join(output_dir, f"grove-dock-v2-{size}-light.png")
        img_light.save(light_path)

        print(f"  {size}px created")

    # Comparison sheet
    print("\nCreating comparison sheet...")
    sheet_w, sheet_h = 900, 500
    sheet = Image.new('RGBA', (sheet_w, sheet_h), light_bg)
    draw = ImageDraw.Draw(sheet)

    # Dark bg on right half
    draw.rectangle([sheet_w//2, 0, sheet_w, sheet_h], fill=dark_bg)

    preview_size = 350
    y = (sheet_h - preview_size) // 2

    # Dark icon on light bg
    img_dark = create_supersampled_v2(preview_size, black, supersample=4)
    x_left = (sheet_w // 4) - (preview_size // 2)
    sheet.paste(img_dark, (x_left, y), img_dark)

    # Light icon on dark bg
    img_light = create_supersampled_v2(preview_size, white, supersample=4)
    x_right = (sheet_w * 3 // 4) - (preview_size // 2)
    sheet.paste(img_light, (x_right, y), img_light)

    comparison_path = os.path.join(output_dir, "grove-dock-v2-comparison.png")
    sheet.save(comparison_path)
    print(f"Comparison: {comparison_path}")

    # Create macOS iconset
    print("\nCreating macOS iconset...")
    iconset_dir = os.path.join(output_dir, "AppIcon.iconset")
    os.makedirs(iconset_dir, exist_ok=True)

    icon_sizes = [16, 32, 64, 128, 256, 512]
    for s in icon_sizes:
        img = create_supersampled_v2(s, black, supersample=4)
        img.save(os.path.join(iconset_dir, f"icon_{s}x{s}.png"))
        if s <= 512:
            img_2x = create_supersampled_v2(s * 2, black, supersample=4)
            img_2x.save(os.path.join(iconset_dir, f"icon_{s}x{s}@2x.png"))

    print(f"Iconset: {iconset_dir}")
    print("Run: iconutil -c icns " + iconset_dir)

    print("\n" + "=" * 50)
    print("Done!")


if __name__ == "__main__":
    main()
