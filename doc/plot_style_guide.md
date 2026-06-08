# Руководство по построению графиков

Эталонный стиль графиков для диплома и отчётов — **HTML-стиль** из [`plots/plot_points.py`](../plots/plot_points.py).  
Все новые графики временных рядов должны повторять этот стиль, а не старые Wolfram/Mathematica-скрипты.

---

## Быстрый старт

```bash
cd scripts
./run_and_plot.sh 4blades          # расчёт + plots/ + копия в doc/images/
```

Только перерисовка без пересчёта (если данные уже есть):

```bash
cd scripts
export CONFIG=4blades
python3 ../plots/plot_points.py
```

Экспорт метрик сразу в `doc/images/`:

```bash
cd scripts
./export_metrics_images.sh 4blades --export-only
```

---

## Канонический визуальный стиль (временные ряды)

Реализован в функции `finalize_html_series_plot()` в [`plots/plot_points.py`](../plots/plot_points.py).

### Обязательные правила

| Элемент | Как делать |
|--------|------------|
| **Заголовок** | **Не рисовать** (`plt.title` не использовать) |
| **Размер фигуры** | `(10, 6)` — константа `HTML_SERIES_FIGSIZE` |
| **Сетка** | Включена: `ax.grid(True, alpha=0.3)` |
| **Цвета серий** | Палитра `matplotlib` `tab10`, по одному цвету на лопатку |
| **Линии** | Только линии (`plt.plot`), **без маркеров** на амплитудах и декременте |
| **Подпись оси Y** | Горизонтально, **сверху слева** (`ax.text` в `transAxes`, `y=1.02`) |
| **Подпись оси X** | Буква **`t`**, справа у конца горизонтальной оси (`ax.annotate`) |
| **Стандартные подписи Y** | `x`, `A`, `δ`, `E`, `Δφ, °` |
| **Легенда** | **Справа от графика**, вне поля данных: `bbox_to_anchor=(1.02, 1.0)` |
| **Отступы** | `subplots_adjust(left=0.08, right=0.76)` — место под легенду |
| **Пределы Y** | Автоматически по данным с отступом ~8% (`auto_plot_ylim`) |
| **DPI при сохранении** | `dpi=200`, `bbox_inches="tight"`, `pad_inches=0.06` |

### Подписи серий в легенде

Не использовать технические имена файлов (`graph_points0`, `amplitude_points1`).

Использовать человекочитаемые подписи:

- **Траектории, амплитуды, декремент:** `лопатка 0`, `лопатка 1`, …
- **Межлопаточная фаза:** `φ(1)−φ(0)`, `φ(2)−φ(1)`, … (следующая минус предыдущая по кольцу)

Функция `blade_label_from_stem()` в `plot_points.py` формирует подпись `лопатка N` из имени файла.

### Минимальный шаблон (Python / matplotlib)

```python
import matplotlib.pyplot as plt
import numpy as np

HTML_SERIES_FIGSIZE = (10, 6)
HTML_LEGEND_RIGHT = 0.76

def finalize_html_series_plot(*, ylabel: str, y_series: list | None = None) -> None:
    ax = plt.gca()
    ax.set_xlabel("")
    ax.set_ylabel("")
    ax.grid(True, alpha=0.3)
    if y_series:
        all_y = np.concatenate([y[np.isfinite(y)] for y in y_series if y.size])
        if all_y.size:
            lo, hi = float(all_y.min()), float(all_y.max())
            pad = max(1.0, 0.08 * (hi - lo))
            ax.set_ylim(lo - pad, hi + pad)
    ax.text(0.0, 1.02, ylabel, transform=ax.transAxes, ha="left", va="bottom", fontsize=12)
    ax.annotate("t", xy=(ax.get_xlim()[1], ax.get_ylim()[0]),
                xycoords="data", xytext=(8, 0), textcoords="offset points",
                ha="left", va="center", fontsize=12)
    ax.legend(loc="upper left", bbox_to_anchor=(1.02, 1.0), frameon=True, fontsize=10)
    plt.gcf().subplots_adjust(left=0.08, right=HTML_LEGEND_RIGHT, top=0.96)

colors = plt.cm.tab10.colors
plt.figure(figsize=HTML_SERIES_FIGSIZE)
for idx, (t, y, label) in enumerate(series):
    plt.plot(t, y, label=label, color=colors[idx % len(colors)])
finalize_html_series_plot(ylabel="A", y_series=[s[1] for s in series])
plt.savefig("amplitudes.png", dpi=200, bbox_inches="tight", pad_inches=0.06)
plt.close()
```

При добавлении новых графиков **предпочтительно** вызывать готовую `finalize_html_series_plot()` из `plot_points.py`, а не копировать логику вручную.

---

## Какие графики строить (single run, `PARALLEL=false`)

Данные: каталог [`wolfram/paramsAndPoints/`](../wolfram/paramsAndPoints/).

| Файл выхода | Источник данных | Подпись Y | Примечание |
|-------------|-----------------|-----------|------------|
| `trajectories.png` | `graph_points{N}.txt` | `x` | Только **подвижные** узлы из конфига |
| `amplitudes.png` | `amplitude_points{N}.txt` | `A` | Без маркеров |
| `decrement_deltas.png` | `decrement_points{N}.txt` | `δ` | Без маркеров |
| `sum_energy.png` | `sumEnergyPoints.txt` | `E` | Одна серия |
| `interblade_phase.png` | `interblade_phase_points{N}.txt` | `Δφ, °` | Межлопаточная разность фаз |

### Что **не** строить в основном наборе

- **`blade_phase.png`** (φ каждой лопатки отдельно) — убран из основного экспорта; для диплома достаточно **межлопаточной** `Δφ`.
- Маркеры на кривых амплитуд и декремента.
- Заголовки внутри PNG (заголовки допустимы только в `index.html`, не на картинках для вставки в текст).

### Автокопирование в `doc/images/`

Скрипт [`scripts/run_and_plot.sh`](../scripts/run_and_plot.sh) после `plot_points.py` копирует в [`doc/images/`](../doc/images/):

- `trajectories.png`
- `amplitudes.png`
- `sum_energy.png`
- `decrement_deltas.png`
- `interblade_phase.png`

---

## Конфигурация и узлы

- Переменная **`CONFIG`** — имя JSON без расширения (например `4blades` → `internal/config/confs/4blades.json`).
- Go-расчёт и Python-графики должны использовать **один и тот же** `CONFIG`.
- Для траекторий и метрик по лопаткам берутся только узлы с `"fixed": false`.

Пример рабочего конфига: [`internal/config/confs/4blades.json`](../internal/config/confs/4blades.json).

---

## Режим sweep (`PARALLEL=true`)

Второй аргумент `run_and_plot.sh`:

```bash
./run_and_plot.sh myconf true
```

При `PARALLEL=true` строятся **карты** по сетке аэрокоэффициентов:

- `decrement_map_node{N}.png` + `decrement_map_node{N}_interactive.html`
- `frequency_map_node{N}.png` + `frequency_map_node{N}_interactive.html`

Источники: `decrementStore{N}.txt`, `freqStore{N}.txt` (три колонки: koef1, koef2, value).

Временные ряды (траектории, амплитуды и т.д.) в этом режиме **не** строятся.

---

## Переменные окружения

| Переменная | По умолчанию | Назначение |
|------------|--------------|------------|
| `CONFIG` | `conf` | Активный конфиг расчёта и графиков |
| `PARALLEL` | `false` | `true` — только карты декремента/частоты |
| `PLOTS_OUTPUT_DIR` | `plots/` | Каталог вывода PNG/HTML (`doc/images` для экспорта в диплом) |
| `PLOT_DARK` | `false` | Тёмный фон PNG (`dark_background`) |
| `A0` | `1` | Коэффициент в аналитической кривой dA/dt |
| `ANALYTICAL_DT_SOURCE` | `peak` | `peak` — Δt из пиков; `integrator` — `times.dt` из конфига |

---

## Специальные графики (отдельные скрипты)

Эти графики **не** используют HTML-стиль; у них свой согласованный стиль для сравнения с аналитикой / таблицами КПА.

### Сравнение numSol и anSol (одна лопатка)

- Скрипт: [`plots/export_damping_comparison.py`](../plots/export_damping_comparison.py)
- Запуск: `scripts/export_damping_comparison.sh`
- Выход: `doc/images/damping_D0.1.png`, `damping_D-0.1.png`, `damping_comparison.png`
- Стиль:
  - численное — **красная** сплошная линия (`#E41A1C`)
  - аналитическое — **чёрный пунктир**
  - легенда: «Численное» / «Аналитическое»

### Погрешность по таблице 1 (КПА)

- Скрипт: [`plots/export_approximation_error.py`](../plots/export_approximation_error.py)
- Запуск: `scripts/export_approximation_error.sh`
- Выход: `doc/images/approximation_error.png`
- Стиль: log-log, подписи `τ` и `err` у концов осей, без сетки (Mathematica-like)

---

## Вспомогательные графики (не для основного текста диплома)

`plot_points.py` также может строить (без HTML-стиля, с обычными `xlabel`/`title`):

- `dsum_energy.png` — производная полной энергии
- `node_energy.png`, `node_d_energy.png` — энергия и dE/dt по узлам
- `analytical_dadt.png` — аналитическая dA/dt по среднему δ
- `d_amplitudes.png` — численная dA/dt (с маркерами)

Их можно оставлять для отладки; в `doc/images/` по умолчанию **не** копируются.

---

## HTML-отчёт

Помимо PNG, `plot_points.py` генерирует [`plots/index.html`](../plots/index.html):

- встраивает все построенные графики;
- показывает сводку декремента из `decrement_details.log`;
- таблицу среднего и СКО межлопаточной фазы;
- формулы расчёта δ;
- переключатель светлой/тёмной темы (только для HTML, не меняет стиль PNG).

---

## Чеклист перед вставкой в диплом

- [ ] График построен через `plot_points.py` или повторяет `finalize_html_series_plot`
- [ ] Нет заголовка на PNG
- [ ] Подпись Y сверху слева, `t` справа внизу
- [ ] Легенда справа, подписи `лопатка N` или `φ(i)−φ(j)`
- [ ] Цвета из `tab10`, сетка с `alpha=0.3`
- [ ] На амплитудах и декременте нет маркеров
- [ ] Файл лежит в `doc/images/` (через `run_and_plot.sh` или `export_metrics_images.sh`)
- [ ] `CONFIG` совпадает с конфигом расчёта в тексте работы

---

## Ссылки на ключевые файлы

| Файл | Роль |
|------|------|
| [`plots/plot_points.py`](../plots/plot_points.py) | Главный генератор графиков и HTML |
| [`scripts/run_and_plot.sh`](../scripts/run_and_plot.sh) | Расчёт + графики + копия в `doc/images/` |
| [`scripts/export_metrics_images.sh`](../scripts/export_metrics_images.sh) | Экспорт метрик в `doc/images/` |
| [`scripts/export_trajectory_images.sh`](../scripts/export_trajectory_images.sh) | Экспорт траекторий в `doc/images/` |
| [`wolfram/paramsAndPoints/`](../wolfram/paramsAndPoints/) | Выходные данные Go-расчёта |
| [`doc/images/`](../doc/images/) | Картинки для диплома |

---

## Для другого чата / агента

Если нужно построить или изменить графики в этом репозитории:

1. Прочитать этот файл: **`doc/plot_style_guide.md`**
2. Не менять стиль временных рядов без явной просьбы пользователя
3. Расширять [`plots/plot_points.py`](../plots/plot_points.py), вызывая `finalize_html_series_plot()`
4. Для диплома выводить PNG в `doc/images/` через `PLOTS_OUTPUT_DIR` или `run_and_plot.sh`
