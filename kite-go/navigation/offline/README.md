# Validation

label|file_f1|file_precision|file_recall|line_f1|line_precision|line_recall
-|:-:|:-:|:-:|:-:|:-:|:-:
angular/angular|0.2412|0.2515|0.3612|0.2142|0.2266|0.3653
apache/airflow|0.255|0.244|0.4616|0.1703|0.195|0.2567
apache/hive|0.2444|0.2582|0.4139|0.1492|0.1568|0.2929
apache/spark|0.2152|0.2175|0.3167|0.1581|0.1594|0.2572
django/django|0.2791|0.2466|0.4652|0.1641|0.1683|0.2464
facebook/react|0.2395|0.2743|0.3581|0.1449|0.1365|0.3267
prestodb/presto|0.2333|0.3225|0.2445|0.1899|0.2035|0.2995
rails/rails|0.2622|0.2148|0.4973|0.1713|0.1765|0.257
spring-projects/spring-framework|0.2298|0.1741|0.4806|0.1611|0.1511|0.2496
tensorflow/tensorflow|0.2081|0.1936|0.3904|0.1341|0.1636|0.2006
mean|0.2408|0.2397|0.3989|0.1657|0.1737|0.2752


# Performance

label|num files|mem allocated (mb)|heap objects (K)|build (s)|serve 5 (ms)|serve 20 (ms)|serve 100 (ms)
-|:-:|:-:|:-:|:-:|:-:|:-:|:-:
angular/angular|7400|30|31|34|167|348|979
apache/airflow|2700|11|21|13|165|330|879
apache/hive|8400|46|33|40|291|654|1696
apache/spark|5200|36|27|25|123|149|625
django/django|3100|23|20|15|115|310|895
facebook/react|1600|19|18|8|53|128|683
kiteco/kiteco|4300|28|29|21|68|98|312
prestodb/presto|6900|25|29|32|102|189|447
rails/rails|3000|23|19|15|61|135|496
spring-projects/spring-framework|7600|27|31|36|118|152|413
tensorflow/tensorflow|12600|67|41|60|324|398|1036
mean|5700|30|27|27|144|262|769
