import multiprocessing

import numpy as np
import matplotlib.pyplot as plt
import copy
import random
import time
from functools import partial
from multiprocessing import Pool
# Strings distance sum value
a = 10
# Step value to change point's coordinates
alpha = 0.1
# h value from formula
h = 1e-6
# Number of points
n = 16
# Generated points range min
range_min = -10
range_max = 10
# Generated points seed
seed =  1000101
# Required precision to end optimization earlier
eps = 1e-4
# Processes count
processes = 12


def generate_points(_n):
    """
    Generates n amount of points in range from min to max by given seed
    :return: generated points array
    """
    points = [[0.0, 0.0]]
    random.seed(seed)
    for i in range(_n):
        x = random.uniform(range_min, range_max)
        y = random.uniform(range_min, range_max)
        points.append([x, y])

    return points


def distance(point1, point2):
    """
    Calculates distance between two given points
    :param point1: point 1
    :param point2: point 2
    :return: distance between points
    """
    return np.sqrt((point1[0] - point2[0]) ** 2 + (point1[1] - point2[1]) ** 2)


def distance_sum(points):
    """
    Calculates distance sum between each point
    :param points: points array
    :return: calculated distance sum
    """
    _sum = 0
    for i, p1 in enumerate(points):
        for p2 in points[i + 1:]:
            _sum += (distance(p1, p2) - a) ** 2

    return _sum


def optimize_points(points):
    """
    Optimizes points location so that strings between points would be near similar distance
    :param points: points array
    :return: optimized points array, current sum of strings distances, iteration count
    """
    global alpha
    points = copy.deepcopy(points)

    max_iterations = 1000
    current_sum = distance_sum(points)
    primary_sum = current_sum.copy()
    counter = 0
    not_improving_counter = 0
    while counter < max_iterations and alpha >= eps and not_improving_counter <= 10:
        counter += 1

        points_gradient = points_gradient_vector(points, current_sum)
        gradient_norm = [item / np.linalg.norm(points_gradient) for item in points_gradient]
        moved_points = move_by_gradient(gradient_norm, points)

        next_sum = distance_sum(moved_points)
        if next_sum < current_sum:
            points = moved_points
            current_sum = next_sum
            not_improving_counter = 0
        else:
            alpha /= 2
            not_improving_counter += 1

    return points, current_sum, counter + 1, primary_sum


def move_by_gradient(gradient_vector, points):
    moved_points = copy.deepcopy(points)

    for i, point in enumerate(moved_points):
        point[0] -= alpha * gradient_vector[i * 2]
        point[1] -= alpha * gradient_vector[i * 2 + 1]

    return moved_points


def points_gradient_vector(points, current_sum):
    """
    Calculates gradient for each point's x and y
    :param points: points array
    :param current_sum: current sum of strings distances
    :return:
    """
    gradients = [0.0, 0.0]
    # Gives work to workers pool
    args = partial(point_gradient, points=points, current_sum=current_sum)
    result = pool.map(args, range(1, len(points)))

    # Converts 2d array to 1d
    [gradients.extend(point) for point in result]

    return gradients


def point_gradient(i, points, current_sum):
    """
    Calculates point given by i and j gradient
    :param i: point index in array
    :param j: x or y of the point
    :param points: point array
    :param current_sum: current value between each strings
    :return: gradient for given point
    """
    # Point's x
    changed_points_x = copy.deepcopy(points)
    changed_points_x[i][0] += h

    # Point's y
    changed_points_y = copy.deepcopy(points)
    changed_points_y[i][1] += h

    return [(distance_sum(changed_points_x) - current_sum) / h, (distance_sum(changed_points_y) - current_sum) / h]


def connect_each_point(points):
    """
    Generates strings connecting each point with each other
    :param points: points
    :return: connected points
    """
    connected = []
    for i, p1 in enumerate(points):
        for p2 in points[i + 1:]:
            connected.append(p1)
            connected.append(p2)

    return connected

# Press the green button in the gutter to run the script.
if __name__ == '__main__':
    # Creates workers pool
    pool = Pool(processes=processes)
    print(multiprocessing.cpu_count())
    # Generates points
    generated_points = generate_points(n)

    # Optimizes points location, measures execution time
    start = time.time()
    optimized_points, sum_value, iterations_count, primary_sum = optimize_points(generated_points)
    end = time.time()
    pool.close()

    # Prints results
    print(
        f"Pradinė kaina: {primary_sum}  Optimizuota kaina: {sum_value} Iteracijų skaičius: {iterations_count}\n Points: {optimized_points}")
    print(f"Vykdymo laikas: {end - start}s")
    connected_points = connect_each_point(optimized_points)

    # Shows primary points data, optimised points, each dot connected graphs
    plt.scatter([x[0] for x in generated_points], [x[1] for x in generated_points], color='b', label='Starting points')
    plt.scatter([x[0] for x in optimized_points], [x[1] for x in optimized_points], color='r', label='Optimized points')
    plt.plot([x[0] for x in connected_points], [x[1] for x in connected_points], color='g',
             label='Optimized points lines')

    plt.legend()
    plt.show()
