import "dart:async";

import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_form.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("dirty draft confirmation $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      await tester.tap(find.text("Nuevo producto"));
      await tester.pumpAndSettle();
      expect(find.text("Identificación del producto"), findsOneWidget);
      expect(find.text("Campos obligatorios marcados con *"), findsOneWidget);
      await tester.enterText(find.byType(EditableText).first, "Borrador");
      await tester.enterText(find.byType(EditableText).last, "BOR-1");
      await tester.tap(find.text("Cancelar borrador"));
      await tester.pumpAndSettle();
      expect(find.text("¿Descartar cambios?"), findsOneWidget);
      await tester.tap(find.text("Cancelar"));
      await tester.pumpAndSettle();
      expect(find.text("Borrador"), findsOneWidget);
      expect(find.text("BOR-1"), findsOneWidget);
      await tester.tap(find.text("Cancelar borrador"));
      await tester.pumpAndSettle();
      await tester.tap(find.text("Descartar cambios"));
      await tester.pumpAndSettle();
      expect(find.byType(EditableText), findsOneWidget);
      expect(find.text("Borrador"), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets("keyboard inset keeps form controls above keyboard", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetViewInsets);
    await tester.pumpWidget(const BalanzaProApp());
    await tester.pumpAndSettle();
    await tester.tap(find.text("Nuevo producto"));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText).first, "Teclado");
    await tester.enterText(find.byType(EditableText).last, "TEC-1");
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key("compact-navigation")), findsNothing);
    final save = find.text("Guardar");
    await tester.scrollUntilVisible(
      save.hitTestable(),
      100,
      scrollable: find
          .descendant(
            of: find.byType(BPForm),
            matching: find.byType(Scrollable),
          )
          .first,
    );
    await tester.pumpAndSettle();
    expect(tester.getBottomRight(save).dy, lessThanOrEqualTo(500));
    await tester.tap(save);
    await tester.pumpAndSettle();
    tester.view.resetViewInsets();
    await tester.pumpAndSettle();
    expect(find.text("Teclado"), findsOneWidget);
    expect(find.byKey(const Key("compact-navigation")), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("required fallback and cancel lock during uncertain submit", (
    tester,
  ) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    final result = Completer<bool>();
    var cancels = 0;
    await tester.pumpWidget(
      BPApp(
        home: BPForm(
          fields: [
            BPTextField(
              id: "name",
              label: "Nombre",
              controller: controller,
              requiredField: true,
            ),
          ],
          onSubmit: (_) => result.future,
          onCancel: () => cancels++,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text("Guardar"));
    await tester.pumpAndSettle();
    expect(find.text("Campo obligatorio"), findsOneWidget);
    await tester.enterText(find.byType(EditableText), "Ejemplo");
    await tester.tap(find.text("Guardar"));
    await tester.pumpAndSettle();
    await tester.tap(find.text("Cancelar"), warnIfMissed: false);
    expect(cancels, 0);
    result.completeError(StateError("uncertain"));
    await tester.pumpAndSettle();
    await tester.tap(find.text("Cancelar"), warnIfMissed: false);
    expect(cancels, 0);
    expect(
      find.text(
        "Resultado sin confirmar. Consulta el estado antes de volver a guardar.",
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });
}
