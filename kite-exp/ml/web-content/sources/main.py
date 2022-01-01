
from sources.web_visualizer.webviewer_generator import WebViewerGenerator

import pandas as pd
from gensim.models import LdaModel



def generate_webviewer():

    pages = pd.read_json("data/pages_with_topic.json", orient='records')
    model = LdaModel.load("data/pages_with_topic_model.lda")
    so_df = pd.read_json("data/SO_top5k_posts.json", orient='index')
    predicted_imps = pd.read_json("data/predicted_kite_impressions_per_so_page.json")
    generator = WebViewerGenerator(pages, model, predicted_imps, so_df)
    generator.generate_topics_viewer()
    generator.generate_so_posts_viewer()




