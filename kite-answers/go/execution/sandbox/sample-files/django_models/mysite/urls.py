from django.conf.urls import include, url

urlpatterns = [
	url(r'^', include('blog.urls', namespace='blog'))
]
