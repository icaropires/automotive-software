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

translate = {
    'vehicleSpeed': 'velocidade',
    'engineRPM': 'frequência do motor',
    'engineCoolanteTemperature': 'temperatura do líquido de arrefecimento',
    'engineLoad': 'carga do motor',
}

fig, axs = plt.subplots(2, 2, figsize=(12, 7), sharex=True, sharey=True)
axs = axs.flat

car_folder = 'cars_output/lab-1-17/'
palettes = ('Oranges', 'Greens', 'RdPu', 'Blues')

for i, param in enumerate(os.listdir(car_folder)):
    file_path = os.path.join(car_folder, param)

    df = pd.read_csv(
        file_path,
        sep=' ',
        names=('date', 'time', param),
        header=None,
        parse_dates={'datetime': [0, 1]},
        date_parser=dateparse
    ).set_index('datetime')

    min_x = datetime(2019, 7, 24, 18, 10, 00)
    max_x = datetime(2019, 7, 24, 18, 26, 20)

    df = df[(df.index >= min_x) & (df.index < max_x)]
    df = df.drop(columns=param)
    df['deltaT'] = df.index.to_series().diff().dt.total_seconds()

    sns.lineplot(palette=palettes[i], legend=False,  data=df, ax=axs[i])

    axs[i].set_xlim(min_x, max_x),
    axs[i].set_title(translate[param], fontsize=16),

    axs[i].set_xlabel('Hora da coleta (tempo)', fontsize=14)
    if not i & 1:
        axs[i].set_ylabel(f'Intervalo entre coletas (s)', fontsize=16)

    axs[i].xaxis.set_major_locator(mdates.MinuteLocator(interval=3))
    axs[i].xaxis.set_major_formatter(mdates.DateFormatter("%H:%M"))
    axs[i].xaxis.set_minor_formatter(mdates.DateFormatter("%H:%M"))

#fig.suptitle('Parâmetros coletados')
#fig.autofmt_xdate()

plt.tight_layout()
plt.show()
