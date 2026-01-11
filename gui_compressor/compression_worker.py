import os
import subprocess
import re
import glob
from PyQt6.QtCore import QThread, pyqtSignal


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
            duration = self.end_time - self.start_time
            if duration <= 0:
                raise ValueError("Invalid duration. End time must be greater than start time.")

            target_size_kb = (self.target_mb * 1024) * 0.95
            total_bitrate = (target_size_kb * 8192) / duration

            video_bitrate = int(total_bitrate - 128000)
            if video_bitrate < 100000:
                video_bitrate = 100000

            pass_log_prefix = self.output_path + "_2pass"

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
                os.devnull,
            ]

            self.run_ffmpeg(cmd_pass1, pass_num=1, total_duration=duration)

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
        # Clear LD_LIBRARY_PATH to prevent PyInstaller's bundled libs from conflicting with system ffmpeg
        env = os.environ.copy()
        env.pop('LD_LIBRARY_PATH', None)
        
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True, env=env)

        time_pattern = re.compile(r"time=(\d{2}):(\d{2}):(\d{2}\.\d+)")

        while True:
            line = process.stderr.readline()
            if not line and process.poll() is not None:
                break

            if line:
                match = time_pattern.search(line)
                if match:
                    h, m, s = match.groups()
                    current_seconds = int(h) * 3600 + int(m) * 60 + float(s)

                    percent_of_pass = (current_seconds / total_duration) * 50
                    total_progress = percent_of_pass + (0 if pass_num == 1 else 50)

                    self.progress_signal.emit(int(total_progress))

        if process.returncode != 0:
            raise Exception(f"FFmpeg Pass {pass_num} failed")
