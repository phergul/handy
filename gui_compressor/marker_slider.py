from PyQt6.QtWidgets import QSlider
from PyQt6.QtCore import pyqtSignal
from PyQt6.QtGui import QPainter, QPen, QColor


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

        h = self.height()
        groove_height = max(6, h // 8)
        groove_top = (h - groove_height) // 2
        groove = self.rect().adjusted(8, groove_top, -8, groove_top + groove_height - h)

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
                if self.end_ms is not None and ms > self.end_ms:
                    ms = self.end_ms
                self.start_ms = ms
            else:
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
