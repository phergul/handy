import sys
from PyQt6.QtWidgets import QApplication
from PyQt6.QtGui import QIcon
from video_app import VideoCompressorApp


if __name__ == "__main__":
    app = QApplication(sys.argv)
    app.setWindowIcon(QIcon("assets/icon.png"))  
    window = VideoCompressorApp()
    window.show()
    sys.exit(app.exec())
