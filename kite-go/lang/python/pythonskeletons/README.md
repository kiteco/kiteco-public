## See curation-team/python-skeletons/README.md for more info
TODO(juan): aliasing types/attrs/submodules -> what if the parent of some node is a child type of some other module? (currently we use the partial path to lookup the type for a given node)
TODO(juan): "aliasing" for arbitraty types, e.g for django.db.models.fields.DateTimeField it basically has same functionality as datetime.datetime,
            currently we simulate this by makeing datetime.datetime a base of DateTimeField... maybe we should make this more explicit and 
            add a way to disable some of this funcitionality?