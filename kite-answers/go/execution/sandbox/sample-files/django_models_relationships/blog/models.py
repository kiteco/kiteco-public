from django.db import models

class Article(models.Model):
    title = models.CharField(max_length=255)

    def __str__(self): 
    	return self.title

class Author(models.Model):
    name = models.CharField(max_length=255)
    articles = models.ManyToManyField(Article)

    def __str__(self): 
    	return self.name