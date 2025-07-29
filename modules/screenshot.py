"""
Screenshot module for MapleStory OCR
Handles capturing screenshots from the game window
"""
from PIL import ImageGrab
import cv2
import numpy as np
import os
from datetime import datetime


def capture_screen_region(rect, region_x, region_y, width, height):
    """Capture a specific region of the screen relative to the window rect
    
    Args:
        rect (tuple): Window rectangle (x1, y1, x2, y2)
        region_x (int): X offset from window left
        region_y (int): Y offset from window top
        width (int): Width of region to capture
        height (int): Height of region to capture
        
    Returns:
        numpy.ndarray: OpenCV image (BGR format)
    """
    x1, y1, x2, y2 = rect
    
    # Calculate absolute screen coordinates of the region
    abs_x = x1 + region_x
    abs_y = y1 + region_y
    
    # Capture the region using PIL
    screenshot = ImageGrab.grab(
        bbox=(abs_x, abs_y, abs_x + width, abs_y + height)
    )
    
    # Convert PIL image to OpenCV format (RGB to BGR)
    return cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)


def save_debug_image(image, name_prefix):
    """Save debug images to temp directory
    
    Args:
        image (numpy.ndarray): OpenCV image to save
        name_prefix (str): Prefix for the filename
        
    Returns:
        numpy.ndarray: The input image (for chaining)
    """
    # Create temp directory structure if it doesn't exist
    temp_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), "temp")
    os.makedirs(temp_dir, exist_ok=True)

    # Add timestamp for unique filenames
    time_suffix = "test"
    # time_suffix = datetime.now().strftime("%H%M%S")
    filename = f"{name_prefix}_{time_suffix}.png"
    save_path = os.path.join(temp_dir, filename)
    
    # Save the image
    cv2.imwrite(save_path, image)
    
    return image
