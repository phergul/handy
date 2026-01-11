import os
import sys
from PyQt6.QtWidgets import QApplication
from PyQt6.QtGui import QIcon
from video_app import VideoCompressorApp


def get_resource_path(relative_path):
    """Get absolute path to resource, works for dev and PyInstaller."""
    try:
        # PyInstaller creates a temp folder and stores path in _MEIPASS
        base_path = sys._MEIPASS
    except Exception:
        base_path = os.path.abspath(".")
    return os.path.join(base_path, relative_path)


if __name__ == "__main__":
    app = QApplication(sys.argv)
    icon_path = get_resource_path("assets/icon.png")
    app.setWindowIcon(QIcon(icon_path))  
    window = VideoCompressorApp()
    window.show()
    sys.exit(app.exec())
