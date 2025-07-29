"""
OCR module for MapleStory OCR
Handles processing images and extracting text using Tesseract OCR
"""
import cv2
import numpy as np
import pytesseract
import re


def preprocess_image(image):
    """Preprocess image to improve OCR accuracy
    
    Args:
        image (numpy.ndarray): Original OpenCV image
        
    Returns:
        numpy.ndarray: Processed image ready for OCR
    """
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


def extract_text(image):
    """Extract text from an image using OCR
    
    Args:
        image (numpy.ndarray): OpenCV image to process
        
    Returns:
        str: Extracted text
    """
    # Preprocess image for better OCR
    processed = preprocess_image(image)
    
    # Configure Tesseract for better accuracy
    custom_config = r"--oem 3 --psm 6"
    
    # OCR the image and return text
    return pytesseract.image_to_string(processed, config=custom_config)


def extract_item_drop_rate(text):
    """Extract Item Drop Rate from OCR text - accumulative
    
    Args:
        text (str): OCR extracted text
        
    Returns:
        int: Item Drop Rate percentage (summed if multiple found, 0 if none found)
    """
    # Look for "Item Drop Rate" pattern in the text
    pattern = r"Item Drop Rate:?\s*\+?(\d+)%"
    matches = re.finditer(pattern, text, re.IGNORECASE)
    
    total_drop_rate = 0
    found_matches = False
    
    for match in matches:
        try:
            total_drop_rate += int(match.group(1))
            found_matches = True
        except (IndexError, ValueError):
            pass
    
    return total_drop_rate


def extract_mesos_obtained(text):
    """Extract Mesos Obtained from OCR text - accumulative
    
    Args:
        text (str): OCR extracted text
        
    Returns:
        int: Mesos Obtained percentage (summed if multiple found, 0 if none found)
    """
    # Look for "Mesos Obtained" pattern in the text - account for optional colon
    pattern = r"Mesos Obtained:?\s*\+?(\d+)%"
    matches = re.finditer(pattern, text, re.IGNORECASE)
    
    total_mesos = 0
    found_matches = False
    
    for match in matches:
        try:
            total_mesos += int(match.group(1))
            found_matches = True
        except (IndexError, ValueError):
            pass
    
    return total_mesos
