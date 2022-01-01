import json

from os import listdir
from os.path import isfile, join
import pandas as pd


def parse_so_moz_datafile(file, result_list):
    content = pd.read_csv(file, names=["Keyword, MinVol, MaxVol, Diff, Rank, URL"], skiprows=1)
    question_id = file.split("/")[-1][:-4]
    for row in content.itertuples():
        url = row[1]
        row = row[0]
        keywords = row[0]
        data = {"question_id": question_id,
                "keywords": keywords,
                "moz_volume": (row[1]+row[2])/2,
                "moz_difficulty": row[3],
                "so_rank": row[4],
                "URL": url
                }
        if keywords in result_list:
            other_data = result_list[keywords]
            if other_data["so_rank"] == data["so_rank"]:
                raise ValueError("2 entry for the same query give the same so_rank for 2 different pages")
            elif other_data["so_rank"] > data["so_rank"]:
                if 'other_results' in other_data:
                    other_data['other_results'].append(data)
                else:
                    other_data['other_results'] = [data]
            else:
                other_results = []
                if 'other_results' in other_data:
                    other_results = other_data['other_results']
                    other_data["other_results"] = None
                other_results.append(other_data)
                data["other_results"] = other_results
                result_list[keywords] = data
        else:
            result_list[keywords] = data
    return len(content)


def parse_moz_files(moz_data_folder, output_file):
    result = {}
    total_count = 0
    files = [join(moz_data_folder, f) for f in listdir(moz_data_folder) if isfile(join(moz_data_folder, f))]

    for f in files:
        total_count += parse_so_moz_datafile(f, result)

    if output_file:
        with open(output_file,  "w") as outfile:
            json.dump(result, outfile)
    return result, total_count
