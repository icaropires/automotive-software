from functools import reduce
from datetime import datetime
import os
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import seaborn as sns

sns.set()

def dateparse(date, time):
    dt = date + " " + time
    return pd.datetime.strptime(dt, '%Y/%m/%d %H:%M:%S.%f')

car_folder = 'cars_output/carro_fernando/'

# Folder must content just the attributes files!
car_df = reduce(
    lambda a, b: a.join(
        b.set_index('datetime'), on='datetime', how='outer'
    ),
    (pd.read_csv(
        os.path.join(car_folder, attr),
        sep=' ',
        names=('date', 'time', attr),
        header=None,
        parse_dates={'datetime': [0, 1]},
        date_parser=dateparse
    ) for attr in os.listdir(car_folder))
).set_index('datetime')

fig, ax = plt.subplots(figsize=(12, 7))

translate = {
    'vehicleSpeed': 'velocidade',
    'engineRPM': 'frequência do motor',
    'engineCoolanteTemperature': 'temperatura do líquido de arrefecimento',
    'engineLoad': 'carga do motor',
}

units = {
    'vehicleSpeed': 'km/h',
    'engineRPM': 'rpm',
    'engineCoolanteTemperature': 'Cº',
    'engineLoad': '%'
}

# Labels from legend must be generated before normalization
labels = tuple(
    '{} (max. {} {})'.format(
        translate.get(c, 'invalid'),
        car_df[c].max(),
        units.get(c, '')
    ) for c in car_df.columns
)

# Normalize
for column in car_df.columns:
    car_df[column] /= car_df[column].max()

sns.lineplot(
    linewidth=1.5,
    data=car_df,
    ax=ax
)

min_x = datetime(2019, 7, 24, 18, 17)
max_x = datetime(2019, 7, 24, 18, 19)
ax.set(
    xlim=(min_x, max_x),
    title='Parâmetros coletados'
)

ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=1))
ax.xaxis.set_major_formatter(mdates.DateFormatter("%H:%M"))
ax.xaxis.set_minor_formatter(mdates.DateFormatter("%H:%M"))

ax.set_yticklabels(
    [str(percent) for percent in
     range(-20, 101, 20)]
)

plt.ylabel('Porcentagem com relação ao máximo atingido (%)', fontsize=15)
plt.xlabel('Hora da coleta (tempo)', fontsize=15)

plt.legend(labels=labels, prop={'size': 12})

plt.show()
