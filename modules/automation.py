"""
Automation module for MapleStory OCR
Handles mouse movements, clicks, and keyboard inputs
"""
import time
import os
import pyautogui
import keyboard
from PIL import ImageGrab
from .window import find_and_activate_maplestory

def click_reroll_button(window_rect):
    """Click the button on the screen to reroll stats
    
    Args:
        window_rect (tuple): Window rectangle (x1, y1, x2, y2)
        
    Returns:
        bool: True if clicking was successful, False otherwise
    """
    # First ensure pyautogui safety
    pyautogui.FAILSAFE = True

    # Calculate click coordinates - adjust these values based on your UI layout
    button_x = window_rect[0] + 647  # Adjust as needed for your game resolution
    button_y = window_rect[1] + 550  # Adjust as needed for your game resolution

    # Click silently at the specified coordinates

    # Re-activate the window to ensure it's in focus
    find_and_activate_maplestory()
    time.sleep(0.1)  # Give window time to come into focus

    # Take debug screenshot before click

    try:
        # Move to button location and click
        pyautogui.moveTo(button_x, button_y, duration=0.3)
        time.sleep(0.05)  # Small delay before click
        pyautogui.click()
        # Click performed

        # Press ENTER twice
        time.sleep(0.05)  # Small delay before key presses
        pyautogui.press("enter")
        time.sleep(0.05)  # Small delay between presses
        pyautogui.press("enter")
        time.sleep(0.05)  # Small delay between presses
        pyautogui.press("enter")
        # Enter keys pressed

        # Success
        time.sleep(0.5)  # Wait for click to register
        
        # Take debug screenshot after click
        return True

    except Exception as e:
        print(f"Error during mouse movement/click: {e}")
        import traceback
        traceback.print_exc()
        return False


def check_stop_key():
    """Check if Ctrl+Esc is pressed
    
    Returns:
        bool: True if stop key combination is pressed, False otherwise
    """
    return keyboard.is_pressed("ctrl+esc")
