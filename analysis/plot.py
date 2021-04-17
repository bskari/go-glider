"""Plot the log data."""

from dataclasses import dataclass, field
from typing import Dict, List, Tuple
import sys

from matplotlib import pyplot

@dataclass
class ParsedData:
    magnetometer: List[Tuple[float, float, float]] = field(default_factory=list)
    accelerometer: List[Tuple[float, float, float]] = field(default_factory=list)


def main() -> None:
    """Plot the log data."""

    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <log-file>")
        return

    with open(sys.argv[1]) as file:
        parsed = parse_file(file)
    print(len(parsed.magnetometer))
    print(len(parsed.accelerometer))


def parse_file(file) -> ParsedData:
    """Parses the data."""

    parsed = ParsedData()

    for line in file:
        pass

    return parsed


if __name__ == "__main__":
    main()
