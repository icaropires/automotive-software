from datetime import datetime
import os
import pandas as pd

def dateparse(date, time):
    dt = date + " " + time
    return pd.datetime.strptime(dt, '%Y/%m/%d %H:%M:%S.%f')

translate = {
    'vehicleSpeed': 'velocidade',
    'engineRPM': 'frequência do motor',
    'engineCoolanteTemperature': 'temperatura do líquido de arrefecimento',
    'engineLoad': 'carga do motor',
}

cars_folder = 'cars_output'

lab1 = dict(mean=0, std=0, count=0)
lab2 = dict(mean=0, std=0, count=0)
real_car = dict(mean=0, std=0, count=0)

min_x = datetime(2019, 7, 24, 18, 27, 00)
max_x = datetime(2019, 7, 24, 18, 43, 20)

for car in os.listdir(cars_folder):
    car_folder = os.path.join(cars_folder, car)

    for param in os.listdir(car_folder):
        file_path = os.path.join(car_folder, param)

        df = pd.read_csv(
            file_path,
            sep=' ',
            names=('date', 'time', param),
            header=None,
            parse_dates={'datetime': [0, 1]},
            date_parser=dateparse
        ).set_index('datetime')

        df = df[(df.index >= min_x) & (df.index < max_x)]
        df_dt = df.index.to_series().diff().dt.total_seconds()

        if car.startswith('lab-1'):
            lab1['count'] += df_dt.count()
            lab1['mean'] += df_dt.mean()
            lab1['std'] += df_dt.std()
        elif car.startswith('lab-2'):
            lab2['count'] += df_dt.count()
            lab2['mean'] += df_dt.mean()
            lab2['std'] += df_dt.std()
        elif car == 'carro_fernando':
            real_car['count'] += df_dt.count()
            real_car['mean'] += df_dt.mean()
            real_car['std'] += df_dt.std()

        break  # just first attribute but easy to modify

print("--------real car----------")
print("count:", real_car['count'])
print("mean:", real_car['mean'])
print("std:", real_car['std'])
print("------------------------")

print("----------lab1----------")
print("count:", lab1['count']/5)
print("mean:", lab1['mean']/5)
print("std:", lab1['std']/5)
print("------------------------")

print("----------lab2----------")
print("count:", lab2['count']/5)
print("mean:", lab2['mean']/5)
print("std:", lab2['std']/5)
print("------------------------")

print("Duration:", max_x - min_x)
