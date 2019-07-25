from datetime import datetime
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import seaborn as sns

sns.set()

def dateparse(date, time):
    dt = date + " " + time
    return pd.datetime.strptime(dt, '%Y/%m/%d %H:%M:%S.%f')

df = pd.read_csv(
    'cars_output/carro_fernando/vehicleSpeed',
    sep=' ',
    names=('date', 'time', 'vehicleSpeed'),
    header=None,
    parse_dates={'datetime': [0, 1]},
    date_parser=dateparse
)

fig, ax = plt.subplots()

sns.lineplot(
    x='datetime',
    y='vehicleSpeed',
    data=df,
    ax=ax
)

min_x = datetime(2019, 7, 24, 18, 7)
ax.set(xlim=(min_x, max(df.datetime)),
       title='Experiment')

ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=2))
ax.xaxis.set_major_formatter(mdates.DateFormatter("%H:%M"))
ax.xaxis.set_minor_formatter(mdates.DateFormatter("%H:%M"))

# plt.figure(figsize=(15, 7))
# plt.legend()
# plt.savefig('my_plog.png')
plt.show()
