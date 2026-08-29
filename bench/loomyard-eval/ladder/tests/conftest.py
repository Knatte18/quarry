"""
Puts the ladder suite's `scripts` directory on `sys.path` so the tests in
this directory import the harness modules (`ladder_config`, `gates`, etc.)
by bare name, without a package/`__init__.py` layout.
"""
import sys
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent / "scripts"

# Guard against stacking duplicate entries if this module is re-imported
# (pytest can re-run conftest collection across multiple invocations).
if str(SCRIPTS_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPTS_DIR))
