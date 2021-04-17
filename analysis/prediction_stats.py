"""Reads in a launch prediction KML and prints out some stats."""
import sys
import math
from dataclasses import dataclass
from typing import List, Tuple


def get_longitude_m_per_d(latitude_d: float) -> float:
    """Returns the distance per degree longitude at a latitude."""
    return math.cos(math.radians(latitude_d)) * 40075


@dataclass
class Point:
    latitude_d: float
    longitude_d: float
    altitude_m: float
    x_offset_m: float
    y_offset_m: float
    seconds: float


def haversine_distance_m(p1: Point, p2: Point) -> float:
    """Returns the distance between 2 points using the Haversine formula."""
    latitude_1_r = math.radians(p1.latitude_d)
    longitude_1_r = math.radians(p1.longitude_d)
    latitude_2_r = math.radians(p2.latitude_d)
    longitude_2_r = math.radians(p2.longitude_d)

    # haversine formula 
    delta_longitude_r = longitude_2_r - longitude_1_r
    delta_latitude_r = latitude_2_r - latitude_1_r
    a = (
        math.sin(delta_latitude_r / 2)**2
        + math.cos(latitude_1_r) * math.cos(latitude_2_r) * math.sin(delta_longitude_r / 2)**2
    )
    c = 2 * math.asin(math.sqrt(a)) 
    r = 6371000  # Radius of earth in meters. Use 3956 for miles
    return c * r


def get_direction_d(p1: Point, p2: Point) -> float:
    """Calculates direction."""
    direction_d = math.degrees(math.atan2(p2.y_offset_m - p1.y_offset_m, p2.x_offset_m - p1.x_offset_m))
    if direction_d < 0.0:
        direction_d += 360.0
    return direction_d


def get_points(csv_file) -> List[Tuple[Point]]:
    """Returns the extraced meter points."""
    found_coordinates = False
    longitude_m_per_d = None
    start_x_m = None
    start_y_m = None

    def extract_points_from_line(line: str) -> Point:
        """Extracts the seconds, x m, y m, altitude m, points from a line."""
        timestamp_s, latitude_d, longitude_d, altitude_m = (float(i) for i in line.rstrip().split(","))
        nonlocal longitude_m_per_d
        nonlocal start_x_m
        nonlocal start_y_m
        if longitude_m_per_d is None:
            longitude_m_per_d = get_longitude_m_per_d(latitude_d)
            start_y_m = latitude_d * 111320
            start_x_m = longitude_d * longitude_m_per_d
        return Point(
            latitude_d,
            longitude_d,
            altitude_m,
            latitude_d * 111320 - start_y_m,
            longitude_d * longitude_m_per_d - start_x_m,
            timestamp_s,
        )

    points = []
    for line in csv_file:
        points.append(extract_points_from_line(line))
    return points


def print_statistics(csv_file) -> None:
    """Prints statistics."""
    points = get_points(csv_file)
    exploded = False
    landed = False

    print("Time, speed m/s, direction d, distance m, altitude m")

    for p1, p2 in zip(points[:-1], points[1:]):
        distance_m = math.sqrt((p2.x_offset_m - p1.x_offset_m) ** 2 + (p2.y_offset_m - p1.y_offset_m) ** 2)
        direction_d = get_direction_d(p1, p2)
        duration_s = p2.seconds - p1.seconds
        speed_m_s = distance_m / duration_s
        time_m = int((p2.seconds - points[0].seconds) / 60)
        time_s = int(p2.seconds - points[0].seconds - time_m * 60)

        if not exploded and p2.altitude_m < p1.altitude_m:
            sys.stdout.write("💥 ")
            exploded = True

        if not landed and p2.altitude_m < points[0].altitude_m:
            distance_from_final_m = haversine_distance_m(p2, points[-1])
            direction_from_final_d = get_direction_d(points[-1], p2)
            print(f"🛬 Landing at {points[0].altitude_m:0.0f}m {p2.latitude_d},{p2.longitude_d}, {distance_from_final_m:0.0f}m {direction_from_final_d:0.0f}° from final point in CSV")
            landed = True

        print(f"{time_m}:{time_s:02},{speed_m_s:0.01f}m/s,{direction_d:0.0f}°,{distance_m:0.01f}m,{p2.altitude_m:0.0f}m")

    print(f"{points[-1].latitude_d}, {points[-1].longitude_d}")


def main() -> None:
    """Main."""
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <path.csv>")
        sys.exit(1)

    with open(sys.argv[1], "r") as file:
        print_statistics(file)


if __name__ == "__main__":
    main()
