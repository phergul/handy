import os
import subprocess
import re
from PyQt6.QtWidgets import (
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
    QSlider,
)
from PyQt6.QtCore import Qt, QUrl
from PyQt6.QtMultimedia import QMediaPlayer, QAudioOutput
from PyQt6.QtMultimediaWidgets import QVideoWidget
from PyQt6.QtGui import QKeySequence

# QShortcut location can vary across PyQt6 builds; try widgets first then gui
try:
    from PyQt6.QtWidgets import QShortcut
except Exception:
    from PyQt6.QtGui import QShortcut

from compression_worker import CompressionWorker
from dropzone import DropZone
from marker_slider import MarkerSlider


class VideoCompressorApp(QWidget):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("MP4 Video Compressor")
        self.resize(900, 900)

        self.current_file_path = None
        self.total_duration = 0.0
        self.output_dir = None

        central_widget = self
        self.layout = QVBoxLayout(central_widget)

        # Drop zone
        self.drop_zone = DropZone(self)
        self.layout.addWidget(self.drop_zone)

        # Video preview
        self.video_widget = QVideoWidget()
        self.video_widget.setMinimumHeight(240)
        self.video_widget.setSizePolicy(QSizePolicy.Policy.Expanding, QSizePolicy.Policy.Expanding)
        try:
            self.video_widget.setAspectRatioMode(Qt.AspectRatioMode.KeepAspectRatio)
        except Exception:
            pass
        self.layout.addWidget(self.video_widget)

        # Player
        self.player = QMediaPlayer()
        self.audio_output = QAudioOutput()
        self.player.setAudioOutput(self.audio_output)
        self.player.setVideoOutput(self.video_widget)

        # Controls
        controls_layout = QHBoxLayout()
        self.btn_play = QPushButton("Play")
        self.btn_play.setEnabled(False)
        self.btn_play.clicked.connect(self.toggle_play)
        controls_layout.addWidget(self.btn_play)

        self.position_slider = MarkerSlider(Qt.Orientation.Horizontal)
        self.position_slider.setEnabled(False)
        self.position_slider.sliderMoved.connect(self.on_slider_moved)
        controls_layout.addWidget(self.position_slider)

        self.lbl_current_time = QLabel("00:00:00.00")
        controls_layout.addWidget(self.lbl_current_time)
        self.layout.addLayout(controls_layout)

        # Volume control
        volume_layout = QHBoxLayout()
        volume_label = QLabel("Volume:")
        volume_layout.addWidget(volume_label)
        
        self.volume_slider = QSlider(Qt.Orientation.Horizontal)
        self.volume_slider.setRange(0, 100)
        self.volume_slider.setValue(100)
        self.volume_slider.setMaximumWidth(150)
        self.volume_slider.valueChanged.connect(self.on_volume_changed)
        volume_layout.addWidget(self.volume_slider)
        
        self.lbl_volume = QLabel("100%")
        self.lbl_volume.setMinimumWidth(40)
        volume_layout.addWidget(self.lbl_volume)
        volume_layout.addStretch()
        self.layout.addLayout(volume_layout)

        # File info and output dir
        self.lbl_filename = QLabel("No file selected")
        fileinfo_layout = QHBoxLayout()
        fileinfo_layout.addWidget(self.lbl_filename)
        self.btn_choose_output = QPushButton("Change Output Dir")
        self.btn_choose_output.setEnabled(False)
        self.btn_choose_output.clicked.connect(self.choose_output_dir)
        fileinfo_layout.addWidget(self.btn_choose_output)
        self.layout.addLayout(fileinfo_layout)
        self.lbl_output_dir = QLabel("")
        self.layout.addWidget(self.lbl_output_dir)

        # Trimming controls
        trim_layout = QHBoxLayout()
        self.spin_start = QDoubleSpinBox()
        self.spin_start.setPrefix("Start: ")
        self.spin_start.setSuffix(" sec")
        self.spin_end = QDoubleSpinBox()
        self.spin_end.setPrefix("End: ")
        self.spin_end.setSuffix(" sec")
        self.spin_end.setMaximum(99999.99)
        trim_layout.addWidget(self.spin_start)
        trim_layout.addWidget(self.spin_end)
        self.spin_start.valueChanged.connect(self.update_markers_from_spins)
        self.spin_end.valueChanged.connect(self.update_markers_from_spins)

        # Small step buttons
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

        self.btn_set_start = QPushButton("Set Start from Player")
        self.btn_set_start.setEnabled(False)
        self.btn_set_start.clicked.connect(self.set_start_from_player)
        trim_layout.addWidget(self.btn_set_start)
        self.btn_set_end = QPushButton("Set End from Player")
        self.btn_set_end.setEnabled(False)
        self.btn_set_end.clicked.connect(self.set_end_from_player)
        trim_layout.addWidget(self.btn_set_end)
        self.layout.addLayout(trim_layout)

        # Compression and output
        comp_layout = QHBoxLayout()
        self.spin_mb = QDoubleSpinBox()
        self.spin_mb.setPrefix("Target: ")
        self.spin_mb.setSuffix(" MB")
        self.spin_mb.setValue(10.0)
        self.spin_mb.setRange(0.1, 10000.0)
        self.spin_mb.setDecimals(2)
        self.spin_mb.setSingleStep(0.5)
        comp_layout.addWidget(self.spin_mb)
        self.layout.addLayout(comp_layout)

        self.txt_output_name = QLineEdit()
        self.txt_output_name.setPlaceholderText("Output Filename (e.g. video_compressed)")
        self.layout.addWidget(self.txt_output_name)

        self.btn_compress = QPushButton("Compress Video")
        self.btn_compress.clicked.connect(self.start_compression)
        self.btn_compress.setEnabled(False)
        self.layout.addWidget(self.btn_compress)

        self.progress_bar = QProgressBar()
        self.progress_bar.setValue(0)
        self.layout.addWidget(self.progress_bar)

        # Signals
        self.player.positionChanged.connect(self.on_position_changed)
        self.player.durationChanged.connect(self.on_duration_changed)
        self.position_slider.marker_changed.connect(self.on_marker_changed)

        QShortcut(QKeySequence("Space"), self).activated.connect(self.toggle_play)
        QShortcut(QKeySequence("S"), self).activated.connect(self.set_start_from_player)
        QShortcut(QKeySequence("E"), self).activated.connect(self.set_end_from_player)

    # -- UI helpers and handlers --
    
    def on_volume_changed(self, value: int):
        volume = value / 100.0
        self.audio_output.setVolume(volume)
        self.lbl_volume.setText(f"{value}%")
    
    def load_video_info(self, file_path):
        self.current_file_path = file_path
        filename = os.path.basename(file_path)
        self.lbl_filename.setText(f"Selected: {filename}")

        try:
            # Check if ffprobe is available
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
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            
            if result.returncode != 0:
                error_msg = result.stderr.strip() if result.stderr else "ffprobe failed"
                raise Exception(f"ffprobe error: {error_msg}")
            
            if not result.stdout.strip():
                raise Exception("ffprobe returned no duration data")
            
            duration = float(result.stdout.strip())

            self.total_duration = duration
            self.spin_end.setValue(duration)
            self.spin_start.setValue(0.0)

            name_no_ext = os.path.splitext(filename)[0]
            self.txt_output_name.setText(f"{name_no_ext}_compressed")

            self.btn_compress.setEnabled(True)
            self.drop_zone.setText(f"Ready: {filename}")

            self.output_dir = os.path.dirname(file_path)
            self.lbl_output_dir.setText(f"Output dir: {self.output_dir}")
            self.btn_choose_output.setEnabled(True)

            try:
                self.player.setSource(QUrl.fromLocalFile(file_path))
            except Exception:
                self.player.setSource(QUrl(file_path))

            self.btn_play.setEnabled(True)
            self.position_slider.setEnabled(True)
            self.btn_set_start.setEnabled(True)
            self.btn_set_end.setEnabled(True)

        except FileNotFoundError:
            QMessageBox.critical(
                self,
                "Missing Dependency",
                "ffprobe (part of ffmpeg) is not installed or not found in PATH.\n\n"
                "Please install ffmpeg:\n"
                "• Linux: sudo apt install ffmpeg (or your package manager)\n"
                "• Windows: Download from ffmpeg.org and add to PATH\n"
                "• macOS: brew install ffmpeg"
            )
            self.lbl_filename.setText(f"Error: ffprobe not found")
            self.drop_zone.setText("Drag & Drop Video Here\n(or click to browse)")
        except Exception as e:
            QMessageBox.warning(
                self,
                "Error Loading Video",
                f"Could not load video information:\n{str(e)}\n\n"
                f"File: {filename}\n\n"
                "Make sure:\n"
                "• The file is a valid video\n"
                "• ffmpeg/ffprobe is installed\n"
                "• You have read permissions for the file"
            )
            self.lbl_filename.setText(f"Error: {str(e)[:50]}")
            self.drop_zone.setText("Drag & Drop Video Here\n(or click to browse)")

    def toggle_play(self):
        if self.player.playbackState() == QMediaPlayer.PlaybackState.PlayingState:
            self.player.pause()
            self.btn_play.setText("Play")
        else:
            self.player.play()
            self.btn_play.setText("Pause")

    def on_position_changed(self, position_ms: int):
        self.position_slider.blockSignals(True)
        self.position_slider.setValue(position_ms)
        self.position_slider.blockSignals(False)
        try:
            self.lbl_current_time.setText(self.format_ms(position_ms))
        except Exception:
            pass

    def on_duration_changed(self, duration_ms: int):
        self.position_slider.setRange(0, max(1, duration_ms))
        self.position_slider.set_duration_ms(duration_ms)
        if self.total_duration == 0 or abs(self.total_duration - (duration_ms / 1000.0)) > 0.5:
            self.total_duration = duration_ms / 1000.0
            self.spin_end.setValue(self.total_duration)
            self.update_markers_from_spins()

    def on_slider_moved(self, position_ms: int):
        self.player.setPosition(position_ms)

    def on_marker_changed(self, start_ms: int, end_ms: int):
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

        output_dir = self.output_dir or os.path.dirname(self.current_file_path)
        output_path = os.path.join(output_dir, output_name)

        self.worker = CompressionWorker(
            self.current_file_path,
            output_path,
            self.spin_start.value(),
            self.spin_end.value(),
            self.spin_mb.value(),
        )

        self.worker.progress_signal.connect(self.progress_bar.setValue)
        self.worker.finished_signal.connect(self.on_compression_finished)
        self.worker.error_signal.connect(self.on_compression_error)

        self.btn_compress.setEnabled(False)
        self.btn_compress.setText("Compressing...")

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
