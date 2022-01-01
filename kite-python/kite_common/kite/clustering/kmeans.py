import numpy as np
from sklearn.cluster import KMeans

def top_down_cluster(data, ids, max_members):
    """
    This function bisects the data set until the size of each cluster is smaller than max_members.

    It returns an array of clusters, and each cluster consists of the index of the data point. 
    """
    clusters = bisect_cluster(data)
    found_clusters = []
    for label in range(clusters.cluster_centers_.shape[0]):
        indices, subset_ids = zip(*[(i, ids[i]) for i, x in enumerate(clusters.labels_) if x == label])
        if len(indices) > max_members: 
            subset = data[indices, :]
            if identical(subset):
                found_clusters.append(subset_ids)
            else:
                found_clusters.extend(top_down_cluster(subset, subset_ids, max_members))
        else:
            found_clusters.append(subset_ids)
    return found_clusters

def cluster(data, ids, num_clusters=30):
    """
    This function uses kmeans clustering to cluster the data set into k clusters. 

    It returns an array of clusters, and each cluster consists of the index of the data point. 
    """
    clusters = KMeans(num_clusters)
    clusters.fit(data)

    found_clusters = []
    for label in range(clusters.cluster_centers_.shape[0]):
        indices, subset_ids = zip(*[(i, ids[i]) for i, x in enumerate(clusters.labels_) if x == label])
        found_clusters.append(subset_ids)
    return found_clusters


def bisect_cluster(data):
    kmeans = KMeans(n_clusters=2)
    kmeans.fit(data)
    return kmeans


def identical(data):
    return all(np.array_equal(d, data[0]) for d in data)
