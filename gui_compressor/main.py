import sys
import os
import subprocess
import re
from PyQt6.QtWidgets import (
    QApplication,
    QMainWindow,
    QWidget,
    QVBoxLayout,
    QLabel,
    QLineEdit,
    QPushButton,
    QProgressBar,
    QFileDialog,
    QHBoxLayout,
    QDoubleSpinBox,
    QSpinBox,
    QMessageBox,
    QSizePolicy,
)
from PyQt6.QtCore import Qt, QThread, pyqtSignal, QUrl
import glob
from PyQt6.QtMultimedia import QMediaPlayer, QAudioOutput
from PyQt6.QtMultimediaWidgets import QVideoWidget
from PyQt6.QtWidgets import QSlider
from PyQt6.QtGui import QPainter, QPen, QColor, QKeySequence
from PyQt6.QtWidgets import QStyleOptionSlider, QStyle

# QShortcut location can vary across PyQt6 builds; try widgets first then gui
try:
    from PyQt6.QtWidgets import QShortcut
except Exception:
    from PyQt6.QtGui import QShortcut


class CompressionWorker(QThread):
    progress_signal = pyqtSignal(int)
    finished_signal = pyqtSignal()
    error_signal = pyqtSignal(str)

    def __init__(self, input_path, output_path, start_time, end_time, target_mb):
        super().__init__()
        self.input_path = input_path
        self.output_path = output_path
        self.start_time = float(start_time)
        self.end_time = float(end_time)
        self.target_mb = int(target_mb)
        self.is_cancelled = False

    def run(self):
        try:
            # 1. Calculate Duration of the TRIMMED clip
            duration = self.end_time - self.start_time
            if duration <= 0:
                raise ValueError(
                    "Invalid duration. End time must be greater than start time."
                )

            # 2. Calculate Bitrate (Ported from your Bash Script)
            # Logic: (TargetMB * 1024 * 0.95) -> KB with 5% buffer
            # Bitrate = (KB * 8192 bits) / duration
            target_size_kb = (self.target_mb * 1024) * 0.95
            total_bitrate = (target_size_kb * 8192) / duration

            # Subtract standard audio bitrate (128k) to give video more room
            # If the calculated bitrate is extremely low, we clamp it to a minimum of 100k
            video_bitrate = int(total_bitrate - 128000)
            if video_bitrate < 100000:
                video_bitrate = 100000

            print(
                f"Calculated Video Bitrate: {video_bitrate / 1000:.2f}k for {duration:.2f}s clip"
            )

            # 3. Define Log Files (Pass log names must match for 2-pass)
            # We use a unique prefix based on the output name to avoid collisions
            pass_log_prefix = self.output_path + "_2pass"

            # 4. PASS 1 (Analysis)
            # -y: overwrite
            # -ss / -to: Trimming positions
            # -an: No audio (not needed for pass 1)
            # -f mp4 /dev/null: Output to nowhere
            cmd_pass1 = [
                "ffmpeg",
                "-y",
                "-i",
                self.input_path,
                "-ss",
                str(self.start_time),
                "-to",
                str(self.end_time),
                "-c:v",
                "libx264",
                "-b:v",
                f"{video_bitrate}",
                "-pass",
                "1",
                "-passlogfile",
                pass_log_prefix,
                "-an",
                "-f",
                "mp4",
                os.devnull,  # Cross-platform /dev/null
            ]

            self.run_ffmpeg(cmd_pass1, pass_num=1, total_duration=duration)

            # 5. PASS 2 (Encoding)
            # -c:a aac -b:a 128k: Re-add audio settings
            cmd_pass2 = [
                "ffmpeg",
                "-y",
                "-i",
                self.input_path,
                "-ss",
                str(self.start_time),
                "-to",
                str(self.end_time),
                "-c:v",
                "libx264",
                "-b:v",
                f"{video_bitrate}",
                "-pass",
                "2",
                "-passlogfile",
                pass_log_prefix,
                "-c:a",
                "aac",
                "-b:a",
                "128k",
                self.output_path,
            ]

            self.run_ffmpeg(cmd_pass2, pass_num=2, total_duration=duration)

            # 6. Cleanup Logs
            # Remove any ffmpeg pass/log artifacts that start with the pass_log_prefix
            try:
                for f in glob.glob(f"{pass_log_prefix}*"):
                    try:
                        os.remove(f)
                    except OSError:
                        pass
            except Exception:
                pass

            self.finished_signal.emit()

        except Exception as e:
            self.error_signal.emit(str(e))

    def run_ffmpeg(self, cmd, pass_num, total_duration):
        """Helper to run ffmpeg and parse progress line-by-line"""
        process = subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True
        )

        # FFmpeg writes progress to stderr
        # Pattern to find time=00:00:05.12
        time_pattern = re.compile(r"time=(\d{2}):(\d{2}):(\d{2}\.\d+)")

        while True:
            line = process.stderr.readline()
            if not line and process.poll() is not None:
                break

            if line:
                # Parse time to calculate percentage
                match = time_pattern.search(line)
                if match:
                    h, m, s = match.groups()
                    current_seconds = int(h) * 3600 + int(m) * 60 + float(s)

                    # Calculate % for this pass
                    percent_of_pass = (current_seconds / total_duration) * 50

                    # Offset: Pass 1 is 0-50%, Pass 2 is 50-100%
                    total_progress = percent_of_pass + (0 if pass_num == 1 else 50)

                    self.progress_signal.emit(int(total_progress))

        if process.returncode != 0:
            raise Exception(f"FFmpeg Pass {pass_num} failed")


class DropZone(QLabel):
    def __init__(self, main_window, parent=None):
        super().__init__(parent)
        self.main_window = main_window
        self.setText("  Drag & Drop Video Here  ")
        self.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.setStyleSheet("""
            QLabel {
                border: 2px dashed #aaa;
                border-radius: 10px;
                font-size: 16px;
                color: #555;
                padding: 30px;
            }
            QLabel:hover {
                border-color: #3daee9; /* KDE Blue */
                background-color: #f0f0f0;
            }
        """)
        self.setAcceptDrops(True)  # ENABLE DRAG & DROP

    def dragEnterEvent(self, event):
        if event.mimeData().hasUrls():
            event.accept()
        else:
            event.ignore()

    def dropEvent(self, event):
        files = [u.toLocalFile() for u in event.mimeData().urls()]
        if files:
            self.main_window.load_video_info(files[0])


class MarkerSlider(QSlider):
    """QSlider subclass that can draw start/end markers relative to duration."""
    marker_changed = pyqtSignal(int, int)

    def __init__(self, orientation, parent=None):
        super().__init__(orientation, parent)
        self.start_ms = None
        self.end_ms = None
        self.duration_ms = 1
        self._dragging_start = False
        self._dragging_end = False

    def set_markers(self, start_ms, end_ms):
        self.start_ms = start_ms
        self.end_ms = end_ms
        self.update()

    def set_duration_ms(self, duration_ms):
        self.duration_ms = max(1, int(duration_ms))
        self.update()

    def paintEvent(self, event):
        super().paintEvent(event)
        painter = QPainter(self)
        painter.setRenderHint(QPainter.RenderHint.Antialiasing)

        # Compute a reasonable groove rect centered vertically in the slider.
        h = self.height()
        groove_height = max(6, h // 8)
        groove_top = (h - groove_height) // 2
        groove = self.rect().adjusted(8, groove_top, -8, groove_top + groove_height - h)

        # Fallback: if rect is empty or duration not set, stop
        if groove.width() <= 0 or self.duration_ms <= 0:
            painter.end()
            return

        def x_for_ms(ms):
            ratio = min(max(ms / float(self.duration_ms), 0.0), 1.0)
            return groove.left() + int(ratio * groove.width())

        if self.start_ms is not None:
            pen = QPen(QColor(0, 180, 0), 2)
            painter.setPen(pen)
            x = x_for_ms(self.start_ms)
            painter.drawLine(x, groove.top(), x, groove.bottom())

        if self.end_ms is not None:
            pen = QPen(QColor(180, 0, 0), 2)
            painter.setPen(pen)
            x = x_for_ms(self.end_ms)
            painter.drawLine(x, groove.top(), x, groove.bottom())

        painter.end()

    def _groove_rect(self):
        h = self.height()
        groove_height = max(6, h // 8)
        groove_top = (h - groove_height) // 2
        groove = self.rect().adjusted(8, groove_top, -8, groove_top + groove_height - h)
        return groove

    def _ms_for_x(self, x):
        groove = self._groove_rect()
        if groove.width() <= 0:
            return 0
        ratio = (x - groove.left()) / float(groove.width())
        ratio = min(max(ratio, 0.0), 1.0)
        return int(ratio * self.duration_ms)

    def mousePressEvent(self, event):
        # Decide if user clicked near start or end marker (8px tolerance)
        x = event.position().x() if hasattr(event, 'position') else event.x()
        groove = self._groove_rect()
        tol = 8
        if self.start_ms is not None:
            sx = groove.left() + int((self.start_ms / float(self.duration_ms)) * groove.width())
            if abs(x - sx) <= tol:
                self._dragging_start = True
                return
        if self.end_ms is not None:
            ex = groove.left() + int((self.end_ms / float(self.duration_ms)) * groove.width())
            if abs(x - ex) <= tol:
                self._dragging_end = True
                return
        super().mousePressEvent(event)

    def mouseMoveEvent(self, event):
        if self._dragging_start or self._dragging_end:
            x = event.position().x() if hasattr(event, 'position') else event.x()
            ms = self._ms_for_x(x)
            if self._dragging_start:
                # ensure start <= end
                if self.end_ms is not None and ms > self.end_ms:
                    ms = self.end_ms
                self.start_ms = ms
            else:
                # dragging end
                if self.start_ms is not None and ms < self.start_ms:
                    ms = self.start_ms
                self.end_ms = ms
            self.update()
            self.marker_changed.emit(self.start_ms or 0, self.end_ms or 0)
            return
        super().mouseMoveEvent(event)

    def mouseReleaseEvent(self, event):
        if self._dragging_start or self._dragging_end:
            self._dragging_start = False
            self._dragging_end = False
            return
        super().mouseReleaseEvent(event)


class VideoCompressorApp(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("MP4 Video Compressor")
        self.resize(900, 900)
        # Use a more square default window so 16:9 previews fit comfortably

        self.current_file_path = None
        self.total_duration = 0.0

        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        self.layout = QVBoxLayout(central_widget)

        # 1. Drop Zone
        self.drop_zone = DropZone(self)
        self.layout.addWidget(self.drop_zone)

        # 1.5 Video Preview
        self.video_widget = QVideoWidget()
        self.video_widget.setMinimumHeight(240)
        # Ensure the video widget expands and keeps aspect ratio when resized
        self.video_widget.setSizePolicy(QSizePolicy.Policy.Expanding, QSizePolicy.Policy.Expanding)
        try:
            # PyQt6: keep aspect ratio where supported
            self.video_widget.setAspectRatioMode(Qt.AspectRatioMode.KeepAspectRatio)
        except Exception:
            pass
        self.layout.addWidget(self.video_widget)

        # Media player
        self.player = QMediaPlayer()
        self.audio_output = QAudioOutput()
        self.player.setAudioOutput(self.audio_output)
        self.player.setVideoOutput(self.video_widget)

        # Playback controls
        controls_layout = QHBoxLayout()
        self.btn_play = QPushButton("Play")
        self.btn_play.setEnabled(False)
        self.btn_play.clicked.connect(self.toggle_play)
        controls_layout.addWidget(self.btn_play)

        self.position_slider = MarkerSlider(Qt.Orientation.Horizontal)
        self.position_slider.setEnabled(False)
        self.position_slider.sliderMoved.connect(self.on_slider_moved)
        controls_layout.addWidget(self.position_slider)

        # Current time label
        self.lbl_current_time = QLabel("00:00:00.00")
        controls_layout.addWidget(self.lbl_current_time)

        self.layout.addLayout(controls_layout)

        # 2. File Info (Hidden by default)
        self.lbl_filename = QLabel("No file selected")
        # Add a small layout for filename + output-dir chooser
        fileinfo_layout = QHBoxLayout()
        fileinfo_layout.addWidget(self.lbl_filename)
        self.btn_choose_output = QPushButton("Change Output Dir")
        self.btn_choose_output.setEnabled(False)
        self.btn_choose_output.clicked.connect(self.choose_output_dir)
        fileinfo_layout.addWidget(self.btn_choose_output)
        self.layout.addLayout(fileinfo_layout)

        self.lbl_output_dir = QLabel("")
        self.layout.addWidget(self.lbl_output_dir)

        # Track chosen output directory
        self.output_dir = None

        # 3. Trimming Controls
        trim_layout = QHBoxLayout()
        self.spin_start = QDoubleSpinBox()
        self.spin_start.setPrefix("Start: ")
        self.spin_start.setSuffix(" sec")
        self.spin_end = QDoubleSpinBox()
        self.spin_end.setPrefix("End: ")
        self.spin_end.setSuffix(" sec")
        self.spin_end.setMaximum(99999.99)  # Allow long videos

        trim_layout.addWidget(self.spin_start)
        trim_layout.addWidget(self.spin_end)
        # Update markers when spin boxes change
        self.spin_start.valueChanged.connect(self.update_markers_from_spins)
        self.spin_end.valueChanged.connect(self.update_markers_from_spins)
        # Small step buttons for fine adjustments
        self.btn_start_minus = QPushButton("-0.5s")
        self.btn_start_minus.clicked.connect(lambda: self.adjust_spin(self.spin_start, -0.5))
        trim_layout.addWidget(self.btn_start_minus)

        self.btn_start_plus = QPushButton("+0.5s")
        self.btn_start_plus.clicked.connect(lambda: self.adjust_spin(self.spin_start, 0.5))
        trim_layout.addWidget(self.btn_start_plus)

        self.btn_end_minus = QPushButton("-0.5s")
        self.btn_end_minus.clicked.connect(lambda: self.adjust_spin(self.spin_end, -0.5))
        trim_layout.addWidget(self.btn_end_minus)

        self.btn_end_plus = QPushButton("+0.5s")
        self.btn_end_plus.clicked.connect(lambda: self.adjust_spin(self.spin_end, 0.5))
        trim_layout.addWidget(self.btn_end_plus)
        # Buttons to capture current playback position
        self.btn_set_start = QPushButton("Set Start from Player")
        self.btn_set_start.setEnabled(False)
        self.btn_set_start.clicked.connect(self.set_start_from_player)
        trim_layout.addWidget(self.btn_set_start)

        self.btn_set_end = QPushButton("Set End from Player")
        self.btn_set_end.setEnabled(False)
        self.btn_set_end.clicked.connect(self.set_end_from_player)
        trim_layout.addWidget(self.btn_set_end)

        self.layout.addLayout(trim_layout)

        # 4. Compression Settings
        comp_layout = QHBoxLayout()
        self.spin_mb = QSpinBox()
        self.spin_mb.setPrefix("Target: ")
        self.spin_mb.setSuffix(" MB")
        self.spin_mb.setValue(10)  # Default
        self.spin_mb.setRange(1, 10000)
        comp_layout.addWidget(self.spin_mb)
        self.layout.addLayout(comp_layout)

        # 5. Output Filename
        self.txt_output_name = QLineEdit()
        self.txt_output_name.setPlaceholderText(
            "Output Filename (e.g. video_compressed)"
        )
        self.layout.addWidget(self.txt_output_name)

        # 6. Action Button & Progress
        self.btn_compress = QPushButton("Compress Video")
        self.btn_compress.clicked.connect(self.start_compression)
        self.btn_compress.setEnabled(False)  # Disabled until file loaded
        self.layout.addWidget(self.btn_compress)

        self.progress_bar = QProgressBar()
        self.progress_bar.setValue(0)
        self.layout.addWidget(self.progress_bar)

    def load_video_info(self, file_path):
        """Runs ffprobe to get duration and updates UI"""
        self.current_file_path = file_path
        filename = os.path.basename(file_path)
        self.lbl_filename.setText(f"Selected: {filename}")

        # Run FFprobe to get duration
        try:
            cmd = [
                "ffprobe",
                "-v",
                "error",
                "-show_entries",
                "format=duration",
                "-of",
                "default=noprint_wrappers=1:nokey=1",
                file_path,
            ]
            result = subprocess.run(cmd, capture_output=True, text=True)
            duration = float(result.stdout.strip())

            self.total_duration = duration
            self.spin_end.setValue(duration)
            self.spin_start.setValue(0.0)

            # Auto-fill output name
            name_no_ext = os.path.splitext(filename)[0]
            self.txt_output_name.setText(f"{name_no_ext}_compressed")

            self.btn_compress.setEnabled(True)
            self.drop_zone.setText(f"Ready: {filename}")
            self.drop_zone.setStyleSheet(
                "border: 2px solid #5cb85c; border-radius: 10px; padding: 30px;"
            )

            # Default output dir to same folder as input unless chosen
            self.output_dir = os.path.dirname(file_path)
            self.lbl_output_dir.setText(f"Output dir: {self.output_dir}")
            self.btn_choose_output.setEnabled(True)

            # Load into media player for preview and enable controls
            try:
                self.player.setSource(QUrl.fromLocalFile(file_path))
            except Exception:
                # Fallback for some PyQt6 versions
                self.player.setSource(QUrl(file_path))
            # Connect player signals
            self.player.positionChanged.connect(self.on_position_changed)
            self.player.durationChanged.connect(self.on_duration_changed)

            # Connect marker slider signal
            self.position_slider.marker_changed.connect(self.on_marker_changed)

            # Keyboard shortcuts
            QShortcut(QKeySequence("Space"), self).activated.connect(self.toggle_play)
            QShortcut(QKeySequence("S"), self).activated.connect(self.set_start_from_player)
            QShortcut(QKeySequence("E"), self).activated.connect(self.set_end_from_player)

            self.btn_play.setEnabled(True)
            self.position_slider.setEnabled(True)
            self.btn_set_start.setEnabled(True)
            self.btn_set_end.setEnabled(True)

        except Exception as e:
            self.lbl_filename.setText(f"Error reading file: {e}")

    def toggle_play(self):
        if self.player.playbackState() == QMediaPlayer.PlaybackState.PlayingState:
            self.player.pause()
            self.btn_play.setText("Play")
        else:
            self.player.play()
            self.btn_play.setText("Pause")

    def on_position_changed(self, position_ms: int):
        # Update slider (position in milliseconds)
        self.position_slider.blockSignals(True)
        self.position_slider.setValue(position_ms)
        self.position_slider.blockSignals(False)

        # Update current time label
        try:
            self.lbl_current_time.setText(self.format_ms(position_ms))
        except Exception:
            pass

    def on_duration_changed(self, duration_ms: int):
        # Set slider range and ensure spin_end agrees when available
        self.position_slider.setRange(0, max(1, duration_ms))
        self.position_slider.set_duration_ms(duration_ms)
        # If ffprobe didn't set duration for some reason, update spin_end
        if self.total_duration == 0 or abs(self.total_duration - (duration_ms / 1000.0)) > 0.5:
            self.total_duration = duration_ms / 1000.0
            self.spin_end.setValue(self.total_duration)
            # refresh markers
            self.update_markers_from_spins()

    def on_slider_moved(self, position_ms: int):
        # Seek player to slider position
        self.player.setPosition(position_ms)

    def on_marker_changed(self, start_ms: int, end_ms: int):
        # Update spin boxes when markers dragged
        try:
            if start_ms is not None:
                self.spin_start.blockSignals(True)
                self.spin_start.setValue(round(start_ms / 1000.0, 2))
                self.spin_start.blockSignals(False)
            if end_ms is not None:
                self.spin_end.blockSignals(True)
                self.spin_end.setValue(round(end_ms / 1000.0, 2))
                self.spin_end.blockSignals(False)
        except Exception:
            pass

    def adjust_spin(self, spin_widget, delta_seconds: float):
        try:
            new_val = max(0.0, spin_widget.value() + delta_seconds)
            spin_widget.setValue(round(new_val, 2))
            self.update_markers_from_spins()
        except Exception:
            pass

    def set_start_from_player(self):
        pos_s = self.player.position() / 1000.0
        self.spin_start.setValue(round(pos_s, 2))
        self.update_markers_from_spins()

    def set_end_from_player(self):
        pos_s = self.player.position() / 1000.0
        self.spin_end.setValue(round(pos_s, 2))
        self.update_markers_from_spins()

    def choose_output_dir(self):
        start_dir = self.output_dir or os.path.dirname(self.current_file_path) if self.current_file_path else os.path.expanduser("~")
        chosen = QFileDialog.getExistingDirectory(self, "Select output directory", start_dir)
        if chosen:
            self.output_dir = chosen
            self.lbl_output_dir.setText(f"Output dir: {self.output_dir}")

    def start_compression(self):
        output_name = self.txt_output_name.text()
        if not output_name.endswith(".mp4"):
            output_name += ".mp4"

        # Determine output folder (use chosen output_dir if set)
        output_dir = self.output_dir or os.path.dirname(self.current_file_path)
        output_path = os.path.join(output_dir, output_name)

        # Setup Worker
        self.worker = CompressionWorker(
            self.current_file_path,
            output_path,
            self.spin_start.value(),
            self.spin_end.value(),
            self.spin_mb.value(),
        )

        # Connect Signals
        self.worker.progress_signal.connect(self.progress_bar.setValue)
        self.worker.finished_signal.connect(self.on_compression_finished)
        self.worker.error_signal.connect(self.on_compression_error)

        # UI State
        self.btn_compress.setEnabled(False)
        self.btn_compress.setText("Compressing...")

        # Start
        self.worker.start()

    def format_ms(self, ms: int) -> str:
        total_s = ms / 1000.0
        h = int(total_s // 3600)
        m = int((total_s % 3600) // 60)
        s = total_s % 60
        return f"{h:02d}:{m:02d}:{s:05.2f}"

    def update_markers_from_spins(self):
        try:
            start_ms = int(self.spin_start.value() * 1000)
            end_ms = int(self.spin_end.value() * 1000)
            self.position_slider.set_markers(start_ms, end_ms)
        except Exception:
            pass

    def on_compression_finished(self):
        self.btn_compress.setEnabled(True)
        self.btn_compress.setText("Compress Video")
        QMessageBox.information(self, "Done", "Video compression complete!")

    def on_compression_error(self, error_msg):
        self.btn_compress.setEnabled(True)
        self.btn_compress.setText("Compress Video")
        self.progress_bar.setValue(0)
        QMessageBox.critical(self, "Error", f"Compression failed:\n{error_msg}")


if __name__ == "__main__":
    app = QApplication(sys.argv)
    window = VideoCompressorApp()
    window.show()
    sys.exit(app.exec())
