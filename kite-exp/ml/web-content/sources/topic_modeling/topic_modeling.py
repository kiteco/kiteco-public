import pandas as pd
import time

from gensim.models import CoherenceModel
from gensim.utils import simple_preprocess
from gensim.parsing.porter import PorterStemmer
import gensim
import gensim.corpora as corpora

from gensim.models.wrappers.ldamallet import malletmodel2ldamodel

from nltk.corpus import stopwords
import nltk


def init_stop_words():
    nltk.download('stopwords')
    stop_words = stopwords.words('english')
    stop_words.extend(['python', 'pyhton', 'pyton', 'pytohn', 'code', 'kite', 'site'])
    stop_words_df = pd.read_json("resources/stop_words.json")
    stop_words.extend(stop_words_df.word.tolist())
    return stop_words


def prepare_dataset(gsc_data, moz_data):
    gsc_data['keywords'] = gsc_data["keys"].apply(lambda k: k[0])
    gsc_data['URL'] = gsc_data['keys'].apply(lambda k: k[1])
    moz_data = moz_data[['keywords', 'URL']]
    gsc_data = gsc_data[['keywords', 'URL']]

    pages = pd.concat([gsc_data, moz_data], axis=0)

    def get_url_features(u):
        keywords = " ".join(u.keywords.tolist())
        return pd.Series(dict(
            keywords=keywords
        ))
    pages = pages.groupby('URL').apply(get_url_features)
    pages['URL'] = pages.index
    return pages


class TopicModeler(object):

    def __init__(self, mallet_path, gsc_data, moz_data, topic_count=1000):
        self.topic_count = topic_count
        self.mallet_path = mallet_path
        self.pages = prepare_dataset(gsc_data, moz_data)
        self.texts, self.corpus, self.id2word = self.build_corpus_and_dictionary()
        self.pages = prepare_dataset(gsc_data, moz_data)
        self.model = self.train_model()
        self.update_pages_with_dominant_topics()

    def prepare_keywords(self):
        stop_words = init_stop_words()
        stemmer = PorterStemmer()

        def preprocess(keywords):
            return [stemmer.stem(word) for word in simple_preprocess(str(keywords)) if word not in stop_words]

        self.pages["pp_content"] = self.pages.keywords.apply(preprocess)

    def build_corpus_and_dictionary(self):
        texts = self.pages.pp_content.values.tolist()
        # Create Dictionary
        id2word = corpora.Dictionary(texts)

        # Create Corpus
        # Term Document Frequency
        corpus = [id2word.doc2bow(text) for text in texts]
        return texts, corpus, id2word

    def train_model(self):
        start = time.time()
        print("Training model with {} topics".format(self.topic_count))
        # Build LDA model
        result = {}
        ldamallet = gensim.models.wrappers.LdaMallet(self.mallet_path, corpus=self.corpus, num_topics=self.topic_count,
                                                     id2word=self.id2word)
        result['mallet_model'] = ldamallet

        conv_mallet = malletmodel2ldamodel(ldamallet)
        result['gensim_model'] = conv_mallet

        coherence_measure = ['u_mass', 'c_v', 'c_uci', 'c_npmi']
        for metric in coherence_measure:
            coherence_model_lda = CoherenceModel(model=conv_mallet, texts=self.texts, dictionary=self.id2word,
                                                 coherence=metric)
            coherence_lda = coherence_model_lda.get_coherence()
            print('\n For {} topics {}: {}'.format(self.topic_count, metric, coherence_lda))
            result[metric] = coherence_lda

        # Compute Coherence Score
        end = time.time()
        print("Computation time : {}s".format(end-start))
        result['time'] = end-start
        return conv_mallet

    def update_pages_with_dominant_topics(self):
        dominant_topics = []
        for i in range(len(self.corpus)):
            topics = self.model[self.corpus[i]]
            if len(topics) > 0:
                s = sorted(topics, key=lambda x: (x[1]), reverse=True)
                dom_top = s[0]

            else:
                dom_top = (-1, 0)
            dominant_topics.append(dom_top)

        dom_topc, ratios = zip(*dominant_topics)
        self.pages["dominant_topic"] = list(dom_topc)
        self.pages["dom_top_ratio"] = list(ratios)

    def get_topic_info(self, topic_id):
        terms = self.model.get_topic_terms(topic_id)
        print("For topic {}".format(topic_id))
        print("Terms:")
        for i in range(10):
            if terms[i][1] < 0.01:
                break
            print("- {} ({:.3f})".format(self.id2word[terms[i][0]], terms[i][1]))
        associated = self.pages[self.pages.dominant_topic == topic_id]
        print("\n{} pages associated to topic {}:".format(len(associated), topic_id))
        if len(associated) > 0:
            sample_size = min(10, len(associated))

            ass_list = associated.sample(sample_size).sort_values('dom_top_ratio', ascending=False).URL.tolist()
            rat_list = associated.sample(sample_size).sort_values('dom_top_ratio', ascending=False)\
                .dom_top_ratio.tolist()

            for s, r in zip(ass_list, rat_list):
                print("- {:.3f} {}".format(r, s))
        else:
            print("No SO pages or Kite examples associated to this topic")

    def print_document(self, i):
        content = self.corpus[i]
        content = sorted(content, key=lambda c: c[1], reverse=True)
        s = " # ".join(["{} ({})".format(self.id2word[idx], count) for idx, count in content])
        print(s)

    def get_page_information(self, url):
        index = self.pages.index.get_loc(url)
        if index == -1:
            print("Can't find a page with this URL")
            return
        row = self.pages.loc[url]
        topics = self.model[self.corpus[index]]
        s = sorted(topics, key=lambda x: (x[1]), reverse=True)
        print("Information for the page {}".format(url))
        print("Impressions {} Clicks {}".format(row["volume"], row["clicks"]))
        print("\nDocument :")
        self.print_document(index)

        print("\n\nTopics : {}".format(s))
        for j in range(len(s)):
            if s[j][1] < .1:
                break
            self.get_topic_info(s[j][0])
        print("\n\n raw keywords: {}".format(row["keywords"]))

    def get_model(self):
        return self.model

    def get_pages_dataset(self):
        return self.pages
