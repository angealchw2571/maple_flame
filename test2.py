import cv2
import numpy as np
import pytesseract
from PIL import ImageGrab
import win32gui
import win32com.client
import win32process
import psutil
import re
import time
import pyautogui
from datetime import datetime
import os
import sys
import keyboard  # Add this at the top with other imports

# Stat types
STR = "STR"
DEX = "DEX"
INT = "INT"
LUK = "LUK"

# Set your main and secondary stats here
MAIN_STAT = DEX  # Change this to your class's main stat (STR/DEX/INT/LUK)
SECONDARY_STAT = STR  # Change this to your class's secondary stat (STR/DEX/INT/LUK)

# Add these near the top with other constants
REROLL_DELAY = 0.1  # Delay between rerolls in seconds
SPLIT_DELAY = 4     # Number of parts to split the delay into for stop key checking

def find_and_activate_maplestory():
    """Find MapleStory process and activate its window"""
    # Find MapleStory process
    maplestory_pid = None
    for proc in psutil.process_iter(["pid", "name"]):
        if proc.info["name"] and proc.info["name"].lower() == "maplestory.exe":
            maplestory_pid = proc.info["pid"]
            break

    if not maplestory_pid:
        raise Exception("MapleStory.exe is not running!")

    # Find the window handle for the MapleStory process
    def callback(hwnd, hwnds):
        if win32gui.IsWindowVisible(hwnd):
            _, pid = win32process.GetWindowThreadProcessId(hwnd)
            if pid == maplestory_pid:
                hwnds.append(hwnd)
        return True

    hwnds = []
    win32gui.EnumWindows(callback, hwnds)

    if not hwnds:
        raise Exception("MapleStory window not found!")

    # Activate the window
    shell = win32com.client.Dispatch("WScript.Shell")
    win32gui.ShowWindow(hwnds[0], 9)  # SW_RESTORE = 9
    shell.SendKeys("%")  # Alt key to focus
    win32gui.SetForegroundWindow(hwnds[0])

    # Small delay to ensure window is active
    time.sleep(0.2)
    return hwnds[0]


def get_maplestory_window():
    """Find and activate the MapleStory window and return its coordinates"""
    hwnd = find_and_activate_maplestory()
    rect = win32gui.GetWindowRect(hwnd)
    return rect


def capture_stat_box(rect, is_before=True):
    """Capture the specific stat box region"""
    x1, y1, x2, y2 = rect

    # The game window is 1366x768
    # Adjust these values based on exact position of the boxes
    box_width = 167  # The blue box appears smaller
    box_height = 104  # The blue box appears smaller

    if is_before:
        # Position for BEFORE box (left side)
        box_x = x1 + 607  # Middle-left of the screen
        box_y = y1 + 350  # About 1/3 down from the top
    else:
        # Position for AFTER box (right side)
        box_x = x1 + 607  # Middle-right of the screen
        box_y = y1 + 495  # Same height as before box

    screenshot = ImageGrab.grab(
        bbox=(box_x, box_y, box_x + box_width, box_y + box_height)
    )
    return cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)


def save_debug_image(image, name_prefix, is_before):
    """Save debug images to temp directory"""
    # Create temp directory structure if it doesn't exist
    temp_dir = os.path.join(os.path.dirname(__file__), "temp")
    os.makedirs(temp_dir, exist_ok=True)

    # Create before/after subdirectories
    stage = "before" if is_before else "after"
    # stage_dir = os.path.join(temp_dir, stage)
    # os.makedirs(stage, exist_ok=True)

    # Save original image
    # time_suffix = datetime.now().strftime("%H%M%S")
    filename = f"{stage}_{name_prefix}.png"
    save_path = os.path.join(temp_dir, filename)
    cv2.imwrite(save_path, image)

    return image


def preprocess_image(image):
    """Preprocess image to improve OCR accuracy"""
    # Convert to grayscale
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)

    # Apply thresholding to get white text on black background
    _, thresh = cv2.threshold(gray, 160, 255, cv2.THRESH_BINARY)

    # Scale up the image
    scaled = cv2.resize(thresh, None, fx=2, fy=2, interpolation=cv2.INTER_CUBIC)

    # Additional processing to enhance text
    kernel = np.ones((2, 2), np.uint8)
    dilated = cv2.dilate(scaled, kernel, iterations=1)

    return dilated


def extract_stats(image, is_before):
    """Extract stats from the image using OCR"""
    # Process and save debug images
    image = save_debug_image(image, "stats", is_before)

    # Preprocess image for better OCR
    processed = preprocess_image(image)

    # Save processed image for debugging
    # stage = "before" if is_before else "after"
    # cv2.imwrite(f"processed_{stage}.png", processed)

    # Configure Tesseract for better accuracy
    custom_config = r"--oem 3 --psm 6"

    # OCR the image
    text = pytesseract.image_to_string(processed, config=custom_config)

    # Print raw OCR text for debugging
    # print(f"\nRaw OCR text for {'before' if is_before else 'after'} image:")
    # print("-------------------")
    # print(text)
    # print("-------------------\n")

    # Initialize stats (only the ones we care about)
    stats = {
        "main_stat": 0,
        "secondary_stat": 0,
        "weapon_attack": 0,
        "magic_attack": 0,
        "all_stat_percent": 0,
    }

    # Process each line of the OCR text
    lines = text.split("\n")
    for line in lines:
        # Remove any spaces around + signs to standardize format
        line = line.replace(" +", "+").replace("+ ", "+")

        # Extract main stat based on MAIN_STAT global variable
        if MAIN_STAT in line:
            try:
                stats["main_stat"] = int(line.split("+")[1])
            except (IndexError, ValueError):
                pass

        # Extract secondary stat based on SECONDARY_STAT global variable
        if SECONDARY_STAT in line:
            try:
                stats["secondary_stat"] = int(line.split("+")[1])
            except (IndexError, ValueError):
                pass

        # Extract Weapon Attack
        if "WEAPON ATTACK" in line or "WEAPON ATT" in line:
            try:
                stats["weapon_attack"] = int(line.split("+")[1])
            except (IndexError, ValueError):
                pass

        # Extract All Stats
        if "All Stats" in line:
            try:
                # Remove the % sign and convert to integer
                value = line.split("+")[1]
                stats["all_stat_percent"] = int(value.replace("%", ""))
            except (IndexError, ValueError):
                pass

    # print("\nFinal stats:", stats)
    return stats


def calculate_flame_score(stats):
    """Calculate the flame score using the formula:
    Main Stat + (Weapon/Magic Attack × 4) + (% All Stat × 8) + (Secondary Stat/8)
    """
    # Calculate each component
    main_stat_value = stats["main_stat"]
    weapon_att_value = stats["weapon_attack"] * 4
    all_stat_value = stats["all_stat_percent"] * 8
    secondary_stat_value = stats["secondary_stat"] / 8

    # Calculate total flame score
    flame_score = (
        main_stat_value + weapon_att_value + all_stat_value + secondary_stat_value
    )

    # print(f"\nFlame Score Breakdown:")
    # print(f"Main Stat ({MAIN_STAT}): {main_stat_value}")
    # print(f"Weapon Attack: {stats['weapon_attack']} → {weapon_att_value}")
    # print(f"All Stat %: {stats['all_stat_percent']}% → {all_stat_value}")
    # print(f"Secondary Stat ({SECONDARY_STAT}): {stats['secondary_stat']} → {secondary_stat_value}")
    # print(f"Total Flame Score: {flame_score}")

    return flame_score


def debug_screenshot(x, y, name, size=20):
    """Take a screenshot of a small area around the specified coordinates"""
    # Calculate the box coordinates (size x size pixels centered on x,y)
    half = size // 2
    box = (x - half, y - half, x + half, y + half)

    # Create temp directory if it doesn't exist
    temp_dir = os.path.join(os.path.dirname(__file__), "temp")
    os.makedirs(temp_dir, exist_ok=True)

    # Take screenshot of the small area
    screenshot = ImageGrab.grab(bbox=box)

    filename = f"{name}.png"
    save_path = os.path.join(temp_dir, filename)
    screenshot.save(save_path)
    print(f"Debug screenshot saved: {filename}")


def reroll(window_rect):
    """Click the 'Use One More' button if it exists"""
    # First ensure pyautogui safety
    pyautogui.FAILSAFE = True
    
    # Calculate click coordinates
    button_x = window_rect[0] + 700  # Align with the stat boxes
    button_y = window_rect[1] + 630  # Below the after box
    
    print(f"Attempting to click at coordinates: ({button_x}, {button_y})")
    
    # Re-activate the window to ensure it's in focus
    hwnd = find_and_activate_maplestory()
    time.sleep(0.1)  # Give window time to come into focus
    
    # Take debug screenshot before click
    debug_screenshot(button_x, button_y, "before_click_debug")
    
    try:
        # Move to button location and click
        pyautogui.moveTo(button_x, button_y, duration=0.3)
        time.sleep(0.05)  # Small delay before click
        pyautogui.click()
        print("Click action performed")
        
        # Press ENTER twice
        time.sleep(0.05)  # Small delay before key presses
        pyautogui.press('enter')
        time.sleep(0.05)  # Small delay between presses
        pyautogui.press('enter')
        print("Enter keys pressed")
        
    except Exception as e:
        print(f"Error during mouse movement/click: {e}")
        import traceback
        traceback.print_exc()
    
    time.sleep(0.5)  # Wait for click to register
    
    # Take debug screenshot after click
    debug_screenshot(button_x, button_y, "after_click_debug")


def print_flame_scores(before_stats, after_stats, before_score, after_score):
    """Print flame scores side by side for better comparison"""
    # Define ANSI color codes
    GREEN = "\033[32m"
    RED = "\033[31m"
    RESET = "\033[0m"

    # Define the width for each side
    width = 40

    # Print header
    print("\n\n\n\n" + "=" * (width * 2 + 3))
    print(f"{'BEFORE':^{width}}|{'AFTER':^{width}}")
    print("=" * (width * 2 + 3))

    # Print main stat with color
    main_stat_diff = after_stats["main_stat"] - before_stats["main_stat"]
    main_stat_color = GREEN if main_stat_diff > 0 else RED if main_stat_diff < 0 else ""
    print(
        f"\nMain Stat ({MAIN_STAT}): {before_stats['main_stat']:<{width-len(str(MAIN_STAT))-14}}|  Main Stat ({MAIN_STAT}): {main_stat_color}{after_stats['main_stat']}{RESET}"
    )

    # Print secondary stat with color
    ss_diff = after_stats["secondary_stat"] - before_stats["secondary_stat"]
    ss_color = GREEN if ss_diff > 0 else RED if ss_diff < 0 else ""
    before_ss = f"Secondary ({SECONDARY_STAT}): {before_stats['secondary_stat']} → {before_stats['secondary_stat'] / 8:.3f}"
    after_ss = f"  Secondary ({SECONDARY_STAT}): {ss_color}{after_stats['secondary_stat']}{RESET} → {ss_color}{after_stats['secondary_stat'] / 8:.3f}{RESET}"
    print(f"{before_ss:<{width}}|{after_ss}")

    # Print weapon attack with color
    wa_diff = after_stats["weapon_attack"] - before_stats["weapon_attack"]
    wa_color = GREEN if wa_diff > 0 else RED if wa_diff < 0 else ""
    before_wa = f"Weapon Attack: {before_stats['weapon_attack']} → {before_stats['weapon_attack'] * 4}"
    after_wa = f"  Weapon Attack: {wa_color}{after_stats['weapon_attack']}{RESET} → {wa_color}{after_stats['weapon_attack'] * 4}{RESET}"
    print(f"{before_wa:<{width}}|{after_wa}")

    # Print all stat with color
    as_diff = after_stats["all_stat_percent"] - before_stats["all_stat_percent"]
    as_color = GREEN if as_diff > 0 else RED if as_diff < 0 else ""
    before_as = f"All Stat %: {before_stats['all_stat_percent']}% → {before_stats['all_stat_percent'] * 8}"
    after_as = f"  All Stat %: {as_color}{after_stats['all_stat_percent']}%{RESET} → {as_color}{after_stats['all_stat_percent'] * 8}{RESET}\n"
    print(f"{before_as:<{width}}|{after_as}")

    # Print divider
    print("-" * (width * 2 + 3))

    # Print total scores with color
    score_diff = after_score - before_score
    score_color = GREEN if score_diff > 0 else RED if score_diff < 0 else ""
    print(
        f"Total Score: {before_score:<{width-13}}|  Total Score: {score_color}{after_score}{RESET}"
    )
    print("=" * (width * 2 + 3))

    # Print score difference with color
    diff = after_score - before_score
    color = GREEN if diff > 0 else RED if diff < 0 else ""
    if diff > 0:
        print(f"\n\nScore Difference: {color}+{diff:.3f}{RESET}\n\n\n")
    else:
        print(f"\n\nScore Difference: {color}{diff:.3f}{RESET}\n\n\n")


def check_stop_key():
    """Check if Ctrl+Esc is pressed"""
    return keyboard.is_pressed('shift+esc')


def main():
    try:
        print("\nPress Ctrl+Esc at any time to stop the script\n")
        
        # Find and activate MapleStory window
        find_and_activate_maplestory()
        window_rect = get_maplestory_window()

        while True:  # Keep running until we get a better score
            # Check for stop key
            if check_stop_key():
                print("\nCtrl+Esc detected. Stopping script...")
                break
                
            # Capture before box
            before_image = capture_stat_box(window_rect, is_before=True)
            before_stats = extract_stats(before_image, is_before=True)
            before_score = calculate_flame_score(before_stats)

            # Capture after box
            after_image = capture_stat_box(window_rect, is_before=False)
            after_stats = extract_stats(after_image, is_before=False)
            after_score = calculate_flame_score(after_stats)

            # Print side by side comparison
            print_flame_scores(before_stats, after_stats, before_score, after_score)

            # Check for stop key again
            if check_stop_key():
                print("\nCtrl+Esc detected. Stopping script...")
                break

            # If after score is lower, reroll
            if after_score < before_score:
                print("\nAfter score is lower. Rerolling...")
                reroll(window_rect)
                
                # Check for stop key during delay
                split_time = REROLL_DELAY / SPLIT_DELAY
                for _ in range(SPLIT_DELAY):  # Split delay into parts
                    if check_stop_key():
                        print("\nCtrl+Esc detected. Stopping script...")
                        return
                    time.sleep(split_time)
            else:
                print("\nGot a better score! Stopping here.")
                break

    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()


def elevate():
    import ctypes
    import win32com.shell.shell as shell

    if ctypes.windll.shell32.IsUserAnAdmin():
        return True

    # If we're not admin, rerun the script with admin privileges
    script = os.path.abspath(sys.argv[0])
    params = ' '.join([script] + sys.argv[1:])
    
    try:
        shell.ShellExecuteEx(lpVerb='runas', lpFile=sys.executable, lpParameters=params)
        sys.exit()
    except Exception as e:
        print(f"Error elevating privileges: {e}")
        return False


if __name__ == "__main__":
    elevate()
    main()
