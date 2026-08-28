"""shapes.py is an impact fixture: it declares a class whose method range is nested inside the
class's own range, so the impact verb's enclosing-symbol selection has an overlap case to resolve
via its greatest-Start tie-break — a case no Go fixture can exercise.
"""


class Shape:
    """Shape is the base geometric shape this fixture's method belongs to."""

    def area(self):
        """area returns the shape's area, a placeholder value in this fixture."""
        return 0
