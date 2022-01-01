import re
import os
import json
import pickle

from spooky import hash64
from numpy import dot
from numpy.linalg import norm

import kite.classification.svm as svm

# suffix of output file for trained model params and feature data
MODEL_FILE_JSON_SUFFIX = "svm_model.json"
MODEL_FILE_PKL_SUFFIX = "svm_model.pkl"
FEAT_DATA_JSON = "feat_data.json"


def add_one_smooth(hash_vec):
    """Adds one count to each feature in a featureVector to
    account for data sparsity.

    Used when constructing the feature vector.
    """

    for i in range(len(hash_vec)):
        hash_vec[i] += 1


def normalize(hash_vec):
    """Normalizes each feature value in the featureVector such that each
    feature is replaced with the fraction of the sum total that it
    occupies, across all the features.
    """

    total = float(sum(hash_vec))

    if total < 1e-8:
        print(
            "Sum total is too small. Could potentially lead to a division by zero error.")
        return

    for i in range(len(hash_vec)):
        hash_vec[i] = (float(hash_vec[i]) / total)


def split_text_by_delims(text):
    """Splits the string by special characters specified.

    Filters out empty strings (as a result of the splitting) before
    returning.
    """

    for ch in [":", ".", "/", "'", "\"",
               "(", ")", "\\", "[", "]", ",", "\t", "\n", "*", "-"]:
        text = text.replace(ch, " ")
    return text.split(" ")


def get_sort_key(item):
    return str(item[1]) + str(item[0])


def compute_most_freq_words_for_file(data_file, n):
    """Computes the most frequently mentioned N_TOP_FREQ_WORDS words in the
    given data_file and return as a listed sorted by descending order of
    frequency.
    """

    encountered = set()
    word_freq = dict()
    with open(data_file) as f:
        for line in f:
            line = line.strip().lower()
            if line in encountered:
                continue
            for word in split_text_by_delims(line):
                word = word.strip()
                if len(word) < 4:
                    continue
                if word in word_freq:
                    word_freq[word] += 1
                else:
                    word_freq[word] = 1

    sorted_words = sorted(word_freq.items(), key=get_sort_key, reverse=True)

    most_freq_words = sorted_words[:n]
    most_freq_words = sorted([tup[0] for tup in most_freq_words])

    return most_freq_words


def compute_char_hash_vec(text, n):
    """Computes new feature vector with features corresponding to the distribution of
    characters in the text.

    Uses the ascii val of each char to map them to a fixed size slice.
    Also, uses add one smoothing to account for sparse data. Finally,
    normalizes each feature value over the entire sum total so each
    feature val corresponds to the percentage of the sum total occupied
    by all characters that got hashed to that index collectively.
    """

    char_hash_vec = [0] * n

    for char in text:
        char = char.strip()
        if len(char) == 0:
            continue
        char_hash_vec[ord(char) % n] += 1

    add_one_smooth(char_hash_vec)
    normalize(char_hash_vec)
    return char_hash_vec


def compute_word_hash_vec(text, n):
    """Computes new feature vector with features corresponding to the distribution of
    words in the text.

    Words are determined by splitting the text by many
    common special characters (see splitTextByDelims()). Uses the hashing
    trick to map all words to a fixed sized slice. Also, uses add one
    smoothing to account for sparse data. Finally, normalizes each feature
    value over the entire sum total so each feature val corresponds to
    the fraction of the sum total occupied by all words that got hashed
    to that index collectively.
    """

    word_hash_vec = [0] * n

    for word in split_text_by_delims(text):
        word = word.strip()
        if len(word) == 0:
            continue

        hashed = hash64(word)
        word_hash_vec[hashed % n] += 1

    add_one_smooth(word_hash_vec)
    normalize(word_hash_vec)

    return word_hash_vec


def compute_special_ch_vec(text):
    feat_vec = []

    if '>' in text:  # prompts with ~ >
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    if '$' in text:  # normal, default prompts
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    if '~' in text:  # prompts with ~ >
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    if '#' in text:  # root prompts tend to end with '#'
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    return feat_vec


def compute_most_freq_words_feature_vec(text, most_freq_words):
    feat_vec = []

    for idx, word in enumerate(most_freq_words):
        if word in text:
            feat_vec.append(1)
        else:
            feat_vec.append(0)

    return feat_vec


def compute_regex_vec(text):
    feat_vec = []

    # prompts generally configured to have timestamps
    timestamp_regex = r'[^:]?\d\d:\d\d:\d\d[^:]?'
    match = re.search(timestamp_regex, text, re.I)
    if match:
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    # prompts sometimes have square brackets and info in them
    parans_regex = r'\[.+\]'
    match = re.search(parans_regex, text, re.I)
    if match:
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    # prompts sometimes (eg. when you log into servers) have something that
    # looks like an email
    email_regex = r'\w+@\w+'
    match = re.search(email_regex, text, re.I)
    if match:
        feat_vec.append(1)
    else:
        feat_vec.append(0)

    return feat_vec


def compute_most_freq_words(files, n):
    """Combines the most frequently occuring words across all the given data files,
    removes duplicate words, and returns a lexicographically sorted list.
    """

    # compute most freq words across files
    most_freq_words = []
    for f in files:
        most_freq_words.extend(compute_most_freq_words_for_file(f, n))

    return sorted(set(most_freq_words))


def feat_vec_to_string(feat_vec):
    result = ""
    for feat in feat_vec:
        result += "{:.5f}".format(feat)
    return result.strip()


def evaluate(model, test_files, output_dir):
    # evaluate model
    for test_file in test_files:
        html_output = evaluate_and_output_pretty_html(
            model,
            test_file)
        with open(os.path.join(output_dir, "output_" + os.path.basename(test_file)), "w+") as f:
            f.write(html_output)


def evaluate_and_output_pretty_html(model, data):
    """Given a data set of feature vectors corresponding to test
    data, use the model to predict the class of each data point.

    Outputs the results in a colored HTML interface for development
    purposes (creates the HTML file automatically).
    """

    encountered = set()

    table = "<table align='center'>"
    total = 0
    correct = 0
    with open(data) as f:
        for line in f:
            if line in encountered:
                continue
            total += 1
            prediction = model.classify(line)
            if prediction == 1.0:
                correct += 1
                table += "<tr style='background-color:#59B100'>" \
                    + "<td style='padding:15px'>" \
                    + line + "</td></tr>"
            else:
                table += "<tr style='background-color:#FB6974; color:white'>" \
                    + "<td style='padding:15px'>" \
                    + line + "</td></tr>"

    table += "</table>"

    html = "<html><body style='font-family:sans-serif'>"
    html += "<h1 align='center'>&#37; identified as errors in file " + \
        os.path.basename(data) + ":<br> " + \
        str(float(correct) / float(total) * 100) + "</h1>"
    html += table
    html += "</body></html>"
    return html


def read_unique_lines(data_file):
    """Loads the data from the given files as lines (unique),
    in preparation for training or testing.
    """

    encountered = set()
    with open(data_file) as f:
        for line in f:
            encountered.add(line)

    return list(encountered)


def load_training_data_with_labels(filename):
    """load_training_data_with_labels loads all the training data
    from one file. The file should be in the following format:
        label\tdata
        label\tdata
        label\tdata
        ...
    """
    all_data = []
    all_labels = []

    with open(filename) as fin:
        for line in fin:
            label, data = line.strip().split('\t')
            all_labels.append(int(label))
            all_data.append(data)

    return all_data, all_labels


def load_training_data(pos_files, neg_files, pos_label, neg_label):
    """Loads training data with one example per line (no labels)."""
    all_data = []
    all_labels = []
    for train_file in pos_files:
        d = read_unique_lines(train_file)
        all_data.extend(d)
        all_labels.extend([pos_label] * len(d))

    for train_file in neg_files:
        d = read_unique_lines(train_file)
        all_data.extend(d)
        all_labels.extend([neg_label] * len(d))

    return all_data, all_labels


def export_model(model, kernel, model_dir):
    if not os.path.isdir(model_dir):
        print("Output dir for model not found.")
        return
    # write model params out to json file
    # cast numpy floats to python floats else cannot dump to json
    dual_coefs = [float(coef) for coef in model.model.dual_coef_[0]]
    support_vecs = [[float(feat) for feat in model.model.support_vectors_[i]]
                    for i in range(len(model.model.support_vectors_))]
    model_params = {
        'kernel_type': kernel,
        'gamma': float(svm.GAMMA),
        'intercept': float(
            model.model.intercept_[0]),
        'dual_coefs': dual_coefs,
        'support_vecs': support_vecs}

    with open(os.path.join(model_dir, kernel + "_" + MODEL_FILE_JSON_SUFFIX), "w+") as model_file:
        json.dump(model_params, model_file)


def save_feature_data(data, model_dir):
    with open(os.path.join(model_dir, FEAT_DATA_JSON), "w+") as feat_data_file:
        json.dump(data, feat_data_file)


def pickle_model(model, kernel, model_dir):
    """pickle_raw_model dumps the raw svm model."""
    filename = os.path.join(model_dir, kernel + "_" + MODEL_FILE_PKL_SUFFIX)
    with open(filename, 'wb') as fout:
        pickle.dump(model, fout)

def get_training_files(input_dir):
    train_files = [
        f for f in os.listdir(input_dir) if os.path.isfile(
            os.path.join(
                input_dir,
                f))]

    pos_files = []
    neg_files = []
    for f in train_files:
        if f.startswith("pos_"):
            pos_files.append(os.path.join(input_dir, f))
        elif f.startswith("neg_"):
            neg_files.append(os.path.join(input_dir, f))

    return pos_files, neg_files


def get_eval_files(input_dir):
    return [os.path.join(input_dir, f) for f in os.listdir(
        input_dir) if os.path.isfile(os.path.join(input_dir, f))]


def cosine_similarity(v1, v2):
    return dot(v1, v2) / (norm(v1) * norm(v2))
