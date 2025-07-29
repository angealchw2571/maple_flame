"""
MapleStory Stats OCR Tool

This script captures a region of the MapleStory window and uses OCR to detect:
- Item Drop Rate
- Mesos Obtained

Usage:
    python dropRate_MesosObt.py         # Stop on first prime line (item drop or mesos)
    python dropRate_MesosObt.py --max   # Search for 2-3 prime lines
"""
import time
import os
import glob
import sys
import shutil
import argparse
from collections import deque
from datetime import datetime
from modules.window import get_maplestory_window
from modules.screenshot import capture_screen_region, save_debug_image
from modules.ocr import extract_text, extract_item_drop_rate, extract_mesos_obtained
from modules.automation import click_reroll_button, check_stop_key

# Define ANSI color codes
GREEN = "\033[32m"
CYAN = "\033[36m"
WHITE = "\033[37m"
RESET = "\033[0m"

# Setup logging
def setup_logging():
    """Setup logging directory and clear temp folder"""
    temp_dir = os.path.join(os.path.dirname(__file__), "temp")
    os.makedirs(temp_dir, exist_ok=True)
    
    # Clear all files in temp directory
    print(f"{CYAN}Clearing temp folder...{RESET}")
    for file_path in glob.glob(os.path.join(temp_dir, "*")):
        try:
            os.remove(file_path)
            print(f"Removed: {os.path.basename(file_path)}")
        except Exception as e:
            print(f"Error removing file {os.path.basename(file_path)}: {e}")
    
    # Create new log file
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    log_file = os.path.join(temp_dir, f"logs_{timestamp}.txt")
    
    return log_file

# Write to log file
def log_ocr_text(log_file, text, stats=None):
    """Write OCR text and stats to log file"""
    with open(log_file, "a", encoding="utf-8") as f:
        f.write(f"\n===== OCR Scan: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')} =====\n")
        f.write(text)
        f.write("\n")
        
        if stats:
            f.write("\nExtracted Stats:\n")
            for key, value in stats.items():
                if key not in ["has_item_keyword", "has_mesos_keyword", "raw_text"]:
                    f.write(f"{key}: {value}\n")
        
        f.write("\n" + "-" * 60 + "\n")


def scan_for_stats():
    """
    Scan for Item Drop Rate and Mesos Obtained stats
    """
    global log_file
    try:
        # Get MapleStory window coordinates silently
        window_rect = get_maplestory_window()
        
        # Define the region to capture - adjust these values as needed for your game resolution
        # These are approximate values - you'll need to adjust them for your specific UI
        region_x = 607  # X coordinate offset from window left
        region_y = 449  # Y coordinate offset from window top
        width = 168     # Width of region to capture
        height = 75    # Height of region to capture
        
        screenshot = capture_screen_region(window_rect, region_x, region_y, width, height)
        
        # Save debug image with timestamp and manage queue
        timestamp = time.strftime("%Y%m%d_%H%M%S")
        save_debug_image(screenshot, f"stat_region_{timestamp}")
        
        # Remove old screenshots if we have more than the queue size
        temp_dir = os.path.join(os.path.dirname(__file__), "temp")
        all_screenshots = sorted(glob.glob(os.path.join(temp_dir, "stat_region_*.png")))
        screenshot_queue_size = 7
        if len(all_screenshots) > screenshot_queue_size:
            for old_screenshot in all_screenshots[:-screenshot_queue_size]:
                try:
                    os.remove(old_screenshot)
                    print(f"Removed old screenshot: {os.path.basename(old_screenshot)}")
                except Exception as e:
                    print(f"Error removing old screenshot: {e}")
        
        # Extract text using OCR
        text = extract_text(screenshot)
        print("\nRaw OCR text:")
        print(f"{CYAN}" + "-" * 40 + f"{RESET}")
        print(text)
        print(f"{CYAN}" + "-" * 40 + f"{RESET}")
        
        # Check for keywords and count prime lines
        text_lower = text.lower()
        has_item_keyword = "item drop" in text_lower
        has_mesos_keyword = "mesos" in text_lower
        
        # Count total prime lines (currently just item drop and mesos)
        prime_line_count = (1 if has_item_keyword else 0) + (1 if has_mesos_keyword else 0)
        
        # Extract specific stats (keeping this for backward compatibility)
        item_drop_rate = extract_item_drop_rate(text)
        mesos_obtained = extract_mesos_obtained(text)
        
        # Display results with color
        print("\nExtracted Stats:")
        print(f"{CYAN}" + "=" * 40 + f"{RESET}")
        
        # Color code based on presence
        item_color = GREEN if item_drop_rate > 0 else WHITE
        mesos_color = GREEN if mesos_obtained > 0 else WHITE
        item_kw_color = GREEN if has_item_keyword else WHITE
        mesos_kw_color = GREEN if has_mesos_keyword else WHITE
        
        print(f"{item_color}Item Drop Rate: +{item_drop_rate}%{RESET}")
        print(f"{mesos_color}Mesos Obtained: +{mesos_obtained}%{RESET}")
        print(f"{CYAN}" + "=" * 40 + f"{RESET}")
        
        # Log the results
        stats = {
            "item_drop_rate": item_drop_rate,
            "mesos_obtained": mesos_obtained,
            "has_item_keyword": has_item_keyword,
            "has_mesos_keyword": has_mesos_keyword,
            "prime_line_count": prime_line_count
        }
        log_ocr_text(log_file, text, stats)
        
        return {
            "item_drop_rate": item_drop_rate,
            "mesos_obtained": mesos_obtained,
            "has_item_keyword": has_item_keyword,
            "has_mesos_keyword": has_mesos_keyword,
            "raw_text": text
        }
        
    except Exception as e:
        print(f"Error: {e}")
        return None




def parse_arguments():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(description="MapleStory Stats OCR Tool")
    parser.add_argument("--max", action="store_true", help="Search for 2-3 prime lines instead of stopping at first one")
    return parser.parse_args()

def main():
    """Main function"""
    # Parse command line arguments
    args = parse_arguments()
    max_mode = args.max
    
    # Initialize text history for stuck detection
    text_history = []
    
    print(f"{CYAN}MapleStory Stats OCR Tool{RESET}")
    print(f"{CYAN}" + "=" * 40 + f"{RESET}")
    print("This tool will capture a region of your MapleStory window")
    print("and extract Item Drop Rate and Mesos Obtained stats.")
    if max_mode:
        print(f"{GREEN}MAX MODE: Searching for 2-3 prime lines{RESET}")
    else:
        print("Standard mode: Stopping on first prime line")
    print(f"{CYAN}" + "=" * 40 + f"{RESET}")
    print("\nPress Ctrl+Esc at any time to exit")
    print("Script will automatically stop if text remains unchanged for 3 consecutive tries")
    
    # Setup logging
    global log_file
    log_file = setup_logging()
    
    try:
        # Find and activate MapleStory window
        window_rect = get_maplestory_window()
        
        # Set up auto-reroll delay
        reroll_delay = 1.0  # seconds between rerolls
        split_delay = 4  # number of parts to split the delay for key checking
        
        while True:
            # Check for stop key combination
            if check_stop_key():
                print("\nCtrl+Esc detected. Exiting...")
                break
                
            # Scan for stats
            print("\nScanning for stats...")
            result = scan_for_stats()
            
            # Add current text to history and keep only last 3
            if result:
                current_text = result['raw_text']
                text_history.append(current_text)
                if len(text_history) > 3:
                    text_history.pop(0)
                
                # Check if text hasn't changed for 3 consecutive tries
                if len(text_history) == 3 and text_history[0] == text_history[1] == text_history[2]:
                    print(f"\n{CYAN}⚠️ OCR text unchanged for 3 consecutive tries. Script might be stuck.{RESET}")
                    print("\nExiting script...")
                    # Log the issue
                    with open(log_file, "a", encoding="utf-8") as f:
                        f.write("\n===== SCRIPT STOPPED: TEXT UNCHANGED FOR 3 TRIES =====\n")
                        f.write(f"Last detected text:\n{current_text}\n")
                    break
            
            if result:
                # Count prime lines found
                prime_lines_found = 0
                if result["has_item_keyword"]:
                    prime_lines_found += 1
                if result["has_mesos_keyword"]:
                    prime_lines_found += 1
                
                # Standard mode: stop on first prime line
                # Max mode: stop only if we have 2+ prime lines
                if (not max_mode and prime_lines_found > 0) or (max_mode and prime_lines_found >= 2):
                    success_message = "Found desired stats!" if not max_mode else f"Found {prime_lines_found} prime lines!"
                    print(f"\n{GREEN}✅ {success_message} Scanning complete.{RESET}")
                    print(f"\n{GREEN}Detected Text:{RESET}\n{result['raw_text']}")
                    
                    # Log the final successful result
                    with open(log_file, "a", encoding="utf-8") as f:
                        success_header = "SUCCESS: DESIRED STATS FOUND" if not max_mode else f"SUCCESS: {prime_lines_found} PRIME LINES FOUND"
                        f.write(f"\n===== {success_header} =====\n")
                        f.write(f"Item Drop Rate: +{result['item_drop_rate']}%\n")
                        f.write(f"Mesos Obtained: +{result['mesos_obtained']}%\n")
                        f.write(f"Total Prime Lines: {prime_lines_found}\n")
                    break
            
            # No desired stats found, click to reroll silently
            click_reroll_button(window_rect)
            
            # Split the delay into parts for responsive key checking
            split_time = reroll_delay / split_delay
            for _ in range(split_delay):
                if check_stop_key():
                    print("\n\033[32mCtrl+Esc detected. Exiting...\033[0m")
                    return
                time.sleep(split_time)
            
    except KeyboardInterrupt:
        print("\n\033[32mExiting...\033[0m")
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    # Create temp directory for debug images if it doesn't exist
    os.makedirs(os.path.join(os.path.dirname(__file__), "temp"), exist_ok=True)
    main()
