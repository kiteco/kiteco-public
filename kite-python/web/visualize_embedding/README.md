This directory contains a visualizer for error messages based on D3.js.

To generate the embedding:

    cd kiteco/kite-python
    PYTHONPATH=. python bin/fit_embedding_by_class.py errormessages.txt embedding.json --gamma -5 --limit 200

To visualize the results:

    cp embedding.json web/visualize_embedding
    cd web/visualize_embedding
    python -c 'python -m http.server'
    open "http://localhost:8000/errorvis.html"

