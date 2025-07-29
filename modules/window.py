"""
Window management module for MapleStory OCR
Handles finding and activating the MapleStory window
"""
import win32gui
import win32com.client
import win32process
import psutil
import time


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
