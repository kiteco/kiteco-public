import setuptools

setuptools.setup(
    name="kite-airflow-dags", # Replace with your own username
    version="0.0.1",
    author="Kite Team",
    description="Kite Airflow codes.",
    packages=setuptools.find_packages(),
    python_requires='>=3.6',
    include_package_data = True,

    entry_points = {
        'airflow.plugins': [
            'google_plugin = kite_airflow.plugins.google:GoogleSheetsPlugin'
        ]
    }
)
