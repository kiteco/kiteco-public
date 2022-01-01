from django.shortcuts import get_object_or_404, render, redirect
from django.utils import timezone

from models import Article

def index(request):
    context = {"articles": Article.objects.all()}
    return render(request, 'blog/index.html', context)

def new(request): 
    return render(request, 'blog/new.html')

def create(request):
    title = request.article['title']
    content = request.article['content']
    
    article = Article(title=title, content=content, pub_date=timezone.now())
    article.save()

    return redirect('blog:index')

def show(request, article_id): 
    article = get_object_or_404(Article, id=article_id)
    context = {"article": article}
    return render(request, 'blog/show.html', context)