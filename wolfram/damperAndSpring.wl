(* ::Package:: *)

(* Начальные условия *)

(* Установить рабочую директорию и создать папку для графиков *)
SetDirectory[DirectoryName[$InputFileName]];
If[!DirectoryQ["Graphs"], CreateDirectory["Graphs"]];

(* Загрузка параметров *)
numParams = ReadList["paramsAndPoints/params.txt", Number]; (* k, m, d, t0, T, x0, v0 *)

t0 = numParams[[4]];
T = numParams[[5]];

substs = <|x0 -> numParams[[-2]], v0 -> numParams[[-1]], m -> numParams[[2]], d -> numParams[[3]], k -> numParams[[1]]|>;

(* Аналитическое решение *)
equation = m x''[t] + d x'[t] + k x[t] == 0;
initConds = {x[0] == x0, x'[0] == v0};
analSol = FullSimplify[DSolveValue[{equation, initConds}, x[t], t]];
analPlot = Plot[analSol /. substs, {t, t0, T}, PlotStyle -> Directive[Yellow, Thickness -> 0.005]];

(* Численное решение *)
numPts = ReadList["paramsAndPoints/points.txt", {Number, Number}];
numPlot = ListLinePlot[numPts, PlotStyle -> Directive[Black, Dashed, Thickness -> 0.005], PlotRange -> All];

result = Show[{analPlot, numPlot}];

Export["Graphs/result.jpeg", result];
