#!/usr/bin/env python3
"""Create side-by-side comparison with reference."""

from PIL import Image
import os

# Load reference
ref_path = "/System/Volumes/Data/Users/iheanyi/Downloads/grove.jpg"
ref = Image.open(ref_path)

# Load our version
our_path = "/Users/iheanyi/development/grove/docs/logo/grove-dock-v2-comparison.png"
our = Image.open(our_path)

# Create comparison
# Reference is wider, let's make them similar heights
target_h = 400

ref_ratio = ref.width / ref.height
ref_new_w = int(target_h * ref_ratio)
ref_resized = ref.resize((ref_new_w, target_h), Image.Resampling.LANCZOS)

our_ratio = our.width / our.height
our_new_w = int(target_h * our_ratio)
our_resized = our.resize((our_new_w, target_h), Image.Resampling.LANCZOS)

# Stack vertically
total_w = max(ref_new_w, our_new_w)
total_h = target_h * 2 + 40

comparison = Image.new('RGB', (total_w, total_h), (255, 255, 255))

# Paste reference at top (centered)
x_ref = (total_w - ref_new_w) // 2
comparison.paste(ref_resized, (x_ref, 10))

# Paste ours at bottom (centered)
x_our = (total_w - our_new_w) // 2
comparison.paste(our_resized.convert('RGB'), (x_our, target_h + 30))

output_path = "/Users/iheanyi/development/grove/docs/logo/reference-vs-ours.png"
comparison.save(output_path)
print(f"Saved: {output_path}")
