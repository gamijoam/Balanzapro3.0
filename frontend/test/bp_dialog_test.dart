import "package:balanzapro_alpha/design_system/bp_dialog.dart";
import "package:balanzapro_alpha/dialog_catalog_main.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/services.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("confirmation only on explicit action at $width", (
      tester,
    ) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      bool? result;
      await tester.pumpWidget(
        BPApp(
          home: Builder(
            builder: (context) => Center(
              child: BPButton(
                label: "Abrir",
                onPressed: () async {
                  result = await BPDialog.confirm(
                    context,
                    title: "Confirmar ajuste",
                    description: "Revisa la operación de ejemplo.",
                    confirmLabel: "Confirmar",
                  );
                },
              ),
            ),
          ),
        ),
      );
      Future<void> open() async {
        await tester.tap(find.text("Abrir"));
        await tester.pumpAndSettle();
        expect(find.text("Confirmar ajuste"), findsOneWidget);
      }

      await open();
      await tester.tap(find.text("Cancelar"));
      await tester.pumpAndSettle();
      expect(result, isFalse);
      await open();
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      await tester.pumpAndSettle();
      expect(result, isFalse);
      expect(find.text("Confirmar ajuste"), findsNothing);
      await open();
      await tester.tapAt(const Offset(4, 4));
      await tester.pumpAndSettle();
      expect(result, isFalse);
      await open();
      await tester.tap(find.text("Confirmar"));
      await tester.pumpAndSettle();
      expect(result, isTrue);
      expect(find.text("Confirmar ajuste"), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets("large text dark dialog and initial keyboard focus cancel", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    bool? result;
    await tester.pumpWidget(
      BPApp(
        home: Builder(
          builder: (context) => Center(
            child: BPButton(
              label: "Abrir",
              onPressed: () async {
                result = await BPDialog.confirm(
                  context,
                  title: "Confirmar eliminación de ejemplo",
                  description: "No se eliminarán datos reales. Revisa antes de continuar.",
                  confirmLabel: "Eliminar ejemplo",
                );
              },
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text("Abrir"));
    await tester.pumpAndSettle();
    expect(find.text("Confirmar eliminación de ejemplo"), findsOneWidget);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(result, isFalse);
    expect(tester.takeException(), isNull);
  });

  testWidgets("catalog and system back never imply a real change", (
    tester,
  ) async {
    await tester.pumpWidget(const BPApp(home: BPDialogExample()));
    await tester.pumpAndSettle();
    await tester.tap(find.text("Abrir confirmación"));
    await tester.pumpAndSettle();
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.text("Cancelado"), findsOneWidget);
    await tester.tap(find.text("Abrir confirmación"));
    await tester.pumpAndSettle();
    await tester.tap(find.text("Confirmar ejemplo"));
    await tester.pumpAndSettle();
    expect(find.text("Confirmado · Sin cambios reales"), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
