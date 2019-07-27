from functools import reduce
from datetime import datetime
import os
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import seaborn as sns
from itertools import cycle

sns.set()


def dateparse(date, time):
    dt = date + " " + time
    return pd.datetime.strptime(dt, '%Y/%m/%d %H:%M:%S.%f')


# Folder must content just the attributes files!
car_folder = '../cars_output/carro_fernando/'

translate = {
    'vehicleSpeed': 'velocidade',
    'engineRPM': 'frequência do motor',
    'engineCoolanteTemperature': 'temp. líquido resfr.',
    'engineLoad': 'carga do motor',
}

units = {
    'vehicleSpeed': 'km/h',
    'engineRPM': 'rpm',
    'engineCoolanteTemperature': 'Cº',
    'engineLoad': '%'
}

fig, axs = plt.subplots(2, 2, figsize=(7,7), sharex=True)
axs = axs.flat

palettes = ( 'Oranges', 'Greens', 'RdPu', 'Blues')

for i, param in enumerate(os.listdir(car_folder)):
    df = pd.read_csv(
        os.path.join(car_folder, param),
        sep=' ',
        names=('date', 'time', param),
        header=None,
        parse_dates={'datetime': [0, 1]},
        date_parser=dateparse,
    ).set_index('datetime')

    sns.lineplot(data=df, palette=palettes[i], legend=False, ax=axs[i])

    min_x = datetime(2019, 7, 24, 18, 10, 00)
    max_x = datetime(2019, 7, 24, 18, 26, 20)

    axs[i].set_xlim(min_x, max_x),
    axs[i].set_xlabel('Hora da coleta (tempo)', fontsize=14)
    axs[i].set_ylabel(f'{translate[param]} ({units[param]})', fontsize=16)

    axs[i].xaxis.set_major_locator(mdates.MinuteLocator(interval=3))
    axs[i].xaxis.set_major_formatter(mdates.DateFormatter("%H:%M"))
    axs[i].xaxis.set_minor_formatter(mdates.DateFormatter("%H:%M"))


#fig.suptitle('Parâmetros coletados')
#fig.autofmt_xdate()

plt.tight_layout()
plt.savefig('my_plot.png')
plt.show()
