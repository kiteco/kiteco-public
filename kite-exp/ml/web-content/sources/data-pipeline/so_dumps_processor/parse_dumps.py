import xml.sax
from xml.sax import ContentHandler
import json
import os
from datetime import date, datetime


class SOPostHandler(ContentHandler):

    def __init__(self):
        self.python_posts = {}
        self.counter = 0
        self.python_counter = 0

    def startElement(self, name, attrs):
        self.counter += 1
        if self.counter % 1000 == 0:
            print("Parsed tag {}, and python posts {}".format(self.counter, self.python_counter))

        if name != "row":
            return
        if attrs["PostTypeId"] != "1":
            return
        if "<python>" not in attrs["Tags"]:
            return

        self.python_counter += 1

        post_id = int(attrs["Id"])
        post = {
            "PostID": post_id,
            "Score": int(attrs["Score"]),
            "ViewCount": int(attrs["ViewCount"]),
            "Title": attrs["Title"],
            "Tags": attrs["Tags"][1:-1].split("><"),
            "AnswerCount": attrs["AnswerCount"],
            "CommentCount": attrs["CommentCount"],
        }
        if "FavoriteCount" in attrs:
            post["FavoriteCount"] = attrs["FavoriteCount"]
        self.python_posts[post_id] = post


class SOPostExtractor(ContentHandler):

    def __init__(self):
        self.counter = 0
        self.python_counter = 0

    def startElement(self, name, attrs):

        self.counter += 1
        if self.counter % 100000 == 0:
            print("Parsed tag {}, and python posts {}".format(self.counter, self.python_counter))
        if name != "row":
            return
        if attrs["PostTypeId"] != "1":
            return
        post_id = int(attrs["Id"])
        path = "/home/moe/workspace/web-content/questions_queries/{}.csv".format(post_id)
        if not os.path.exists(path):
            return
        self.python_counter += 1
        target_path = "/home/moe/workspace/web-content/questions_content/{}.txt".format(post_id)
        with open(target_path, "w") as outfile:
            outfile.write(attrs["Body"])


def _parse_so_dump(target):
    parser = xml.sax.make_parser()
    parser.setFeature(xml.sax.handler.feature_namespaces, 0)
    handler = SOPostHandler()
    parser.setContentHandler(handler)
    parser.parse(target)
    return handler.python_posts

def _add_view_count(recent_data, old_data, day_diff, normal_day_count=30):
    for k in recent_data:
        old_count = 0
        if k in old_data:
            old_count = old_count[k]["ViewCount"]
        new_count = recent_data[k]["ViewCount"]
        recent_data[k]["newViews"] = int((new_count - old_count)*normal_day_count/day_diff)


def extract_SO_meta_informations(so_dump_file, so_dump_date, target_file, so_dump_previous_month=None, so_dump_previous_month_date=None):
    """
    Parse SO dump file and extract the meta information of posts
    If a second dump is provided, the delta of the number of post is
    :param so_dump_file: File to parse to extract post meta information
    :param so_dump_date: Date of the first dump (format YYYY/MM/DD)
    :param target_file: File to write the result (json dict format)
    :param so_dump_previous_month: Optional second file to parse to get a delta of view count
    :param day_diff: Number of day to normalize the view count delta (normalized to 30 days)
    """

    recent_data = _parse_so_dump(so_dump_file)
    if so_dump_previous_month:
        if not so_dump_previous_month_date:
            raise ValueError("Please provide the date of the previous SO dump. Can't normalize the view count without it")
        date_format = "%Y/%m/%d"
        recent = datetime.strptime(so_dump_date, date_format)
        old = datetime.strptime(so_dump_previous_month_date, date_format)
        delta_day = (recent - old).days
        old_data = _parse_so_dump(so_dump_previous_month)
        _add_view_count(recent_data, old_data, delta_day)

    with open(target_file, "w") as outfile:
        json.dump(outfile, recent_data)



if ( __name__ == "__main__"):


    SO_dump_march_4th = "/data/kite/SO_dumps/Posts_march-2019.xml"
    # with open("/data/kite/SO_dumps/python_posts_april.json", "w") as outfile:
    #    json.dump(handler.python_posts, outfile)

