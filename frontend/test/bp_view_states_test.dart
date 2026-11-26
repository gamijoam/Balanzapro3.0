import "package:balanzapro_alpha/design_system/bp_view_states.dart";
import "package:balanzapro_alpha/states_catalog_main.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("state presentation and actions at $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      var retries = 0;
      Future<void> show(Widget child) async {
        await tester.pumpWidget(BPApp(home: Center(child: child)));
        await tester.pumpAndSettle();
      }

      await show(const BPEmptyState());
      expect(find.text("No hay resultados"), findsOneWidget);
      expect(find.byType(BPButton), findsNothing);
      await show(BPErrorState(onRetry: () => retries++));
      await tester.tap(find.text("Reintentar"));
      expect(retries, 1);
      await show(const BPOfflineState());
      expect(find.text("Sin conexión"), findsOneWidget);
      expect(find.byType(BPButton), findsNothing);
      await show(const BPLoadingState());
      expect(find.byType(BPSkeleton), findsNWidgets(2));
      expect(find.text("Cargando registros"), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets("dark 200 percent text and optional retry", (tester) async {
    tester.view.physicalSize = const Size(360, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    var calls = 0;
    await tester.pumpWidget(
      BPApp(
        home: ListView(children: [BPOfflineState(onRetry: () => calls++)]),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text("Comprobar conexión"));
    expect(calls, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets("catalog fixture action never fabricates recovery", (
    tester,
  ) async {
    await tester.pumpWidget(const BPApp(home: BPStatesExample()));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text("Comprobar conexión").hitTestable(),
      250,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(find.text("Comprobar conexión").hitTestable(), findsOneWidget);
    await tester.tap(find.text("Comprobar conexión"));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text("Acciones de ejemplo: 1").hitTestable(),
      -250,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.text("Acciones de ejemplo: 1"), findsOneWidget);
    expect(find.text("Conexión recuperada"), findsNothing);
    expect(tester.takeException(), isNull);
  });
}
