import os
from PyQt6.QtWidgets import QLabel, QFileDialog
from PyQt6.QtCore import Qt, QUrl
from PyQt6.QtGui import QCursor


class DropZone(QLabel):
    def __init__(self, main_window, parent=None):
        super().__init__(parent)
        self.main_window = main_window
        self.setText("  Drag & Drop Video Here\n(or click to browse)  ")
        self.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.setStyleSheet("""
            QLabel {
                border: 2px dashed #aaa;
                border-radius: 10px;
                font-size: 16px;
                color: #555;
                padding: 30px;
                cursor: pointer;
            }
            QLabel:hover {
                border-color: #3daee9; /* KDE Blue */
                background-color: #f0f0f0;
            }
        """)
        self.setAcceptDrops(True)
        self.setCursor(QCursor(Qt.CursorShape.PointingHandCursor))
    
    def mousePressEvent(self, event):
        """Open file dialog when dropzone is clicked."""
        file_path, _ = QFileDialog.getOpenFileName(
            self.main_window,
            "Select a video file",
            os.path.expanduser("~"),
            "Video Files (*.mp4 *.avi *.mov *.mkv *.flv *.wmv *.webm);;All Files (*)"
        )
        if file_path:
            self.main_window.load_video_info(file_path)

    def dragEnterEvent(self, event):
        # Accept drag when URLs or text/uri-list are present
        if event.mimeData().hasUrls() or event.mimeData().hasText():
            event.acceptProposedAction()
        else:
            event.ignore()

    def dragMoveEvent(self, event):
        # Some file managers require accepting drag move for DnD to work
        if event.mimeData().hasUrls() or event.mimeData().hasText():
            event.acceptProposedAction()
        else:
            event.ignore()

    def dropEvent(self, event):
        files = []

        if event.mimeData().hasUrls():
            files = [u.toLocalFile() for u in event.mimeData().urls()]
        elif event.mimeData().hasText():
            # Fallback: parse text/uri-list or plain paths
            text = event.mimeData().text().strip()
            for line in text.splitlines():
                line = line.strip()
                if not line:
                    continue
                # Handle file:// URIs
                if line.startswith("file://"):
                    url = QUrl(line)
                    path = url.toLocalFile()
                else:
                    path = line
                if os.path.isfile(path):
                    files.append(path)

        if files:
            event.acceptProposedAction()
            self.main_window.load_video_info(files[0])
        else:
            event.ignore()
    
    def enterEvent(self, event):
        """Update cursor on hover."""
        self.setCursor(QCursor(Qt.CursorShape.PointingHandCursor))
        super().enterEvent(event)
