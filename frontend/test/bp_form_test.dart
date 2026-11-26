import "dart:async";

import "package:balanzapro_alpha/design_system/catalog/form_example.dart";

import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:balanzapro_alpha/design_system/bp_form.dart";
import "package:flutter/widgets.dart";
import "package:flutter/services.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("form validates and sends once at $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final result = Completer<bool>();
      final name = TextEditingController(), code = TextEditingController();
      addTearDown(name.dispose);
      addTearDown(code.dispose);
      var calls = 0;
      Map<String, String>? sent;
      await tester.pumpWidget(
        BPApp(
          home: BPForm(
            fields: [
              BPTextField(
                id: "name",
                label: "Nombre",
                controller: name,
                validate: (v) => v.trim().isEmpty ? "Escribe un nombre" : null,
              ),
              BPTextField(
                id: "code",
                label: "Código",
                controller: code,
                validate: (v) => v.trim().isEmpty ? "Escribe un código" : null,
              ),
            ],
            onSubmit: (v) {
              calls++;
              sent = v;
              return result.future;
            },
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      expect(find.text("Escribe un nombre"), findsOneWidget);
      expect(find.text("Escribe un código"), findsOneWidget);
      expect(
        tester
            .widget<EditableText>(find.byType(EditableText).first)
            .focusNode
            .hasFocus,
        isTrue,
      );
      expect(calls, 0);
      await tester.enterText(find.byType(EditableText).first, "Arroz");
      await tester.enterText(find.byType(EditableText).last, "ALI-1");
      await tester.pumpAndSettle();
      expect(find.text("Escribe un nombre"), findsNothing);
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      expect(find.text("Guardando…"), findsOneWidget);
      await tester.tap(find.text("Guardando…"), warnIfMissed: false);
      expect(calls, 1);
      expect(sent, {"name": "Arroz", "code": "ALI-1"});
      result.complete(false);
      await tester.pumpAndSettle();
      expect(
        find.text("No se pudo guardar. Intenta nuevamente."),
        findsOneWidget,
      );
      expect(name.text, "Arroz");
      expect(code.text, "ALI-1");
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets("keyboard next and done; successful callback", (tester) async {
    final a = TextEditingController(text: "Uno"),
        b = TextEditingController(text: "Dos");
    addTearDown(a.dispose);
    addTearDown(b.dispose);
    var calls = 0;
    await tester.pumpWidget(
      BPApp(
        home: BPForm(
          fields: [
            BPTextField(id: "a", label: "Primero", controller: a),
            BPTextField(id: "b", label: "Segundo", controller: b),
          ],
          onSubmit: (_) async {
            calls++;
            return true;
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(EditableText).first);
    await tester.testTextInput.receiveAction(TextInputAction.next);
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<EditableText>(find.byType(EditableText).last)
          .focusNode
          .hasFocus,
      isTrue,
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(calls, 1);
    expect(find.text("Cambios guardados"), findsOneWidget);
  });

  testWidgets(
    "confirmed rejection permits retry and unknown result blocks retry",
    (tester) async {
      final c = TextEditingController(text: "Ejemplo");
      addTearDown(c.dispose);
      var calls = 0;
      await tester.pumpWidget(
        BPApp(
          home: BPForm(
            fields: [BPTextField(id: "name", label: "Nombre", controller: c)],
            onSubmit: (_) async {
              calls++;
              if (calls == 1) return false;
              throw StateError("private transport detail");
            },
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      expect(
        find.text("No se pudo guardar. Intenta nuevamente."),
        findsOneWidget,
      );
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      expect(
        find.text(
          "Resultado sin confirmar. Consulta el estado antes de volver a guardar.",
        ),
        findsOneWidget,
      );
      expect(find.textContaining("private transport detail"), findsNothing);
      await tester.tap(find.text("Guardar"), warnIfMissed: false);
      await tester.pumpAndSettle();
      expect(calls, 2);
      expect(c.text, "Ejemplo");
    },
  );

  testWidgets("form with enlarged text in dark mode", (tester) async {
    tester.view.physicalSize = const Size(360, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    final c = TextEditingController();
    addTearDown(c.dispose);
    await tester.pumpWidget(
      BPApp(
        home: BPForm(
          fields: [
            BPTextField(
              id: "long",
              label: "Nombre completo del producto",
              controller: c,
              validate: (_) => "Escribe el nombre completo del producto",
            ),
          ],
          onSubmit: (_) async => false,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text("Guardar"));
    await tester.pumpAndSettle();
    expect(
      find.text("Escribe el nombre completo del producto"),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets("catalog example explicitly has no persistence", (tester) async {
    await tester.pumpWidget(const BPApp(home: BPFormExample()));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText).first, "Ejemplo");
    await tester.enterText(find.byType(EditableText).last, "SKU-001");
    await tester.tap(find.text("Guardar"));
    await tester.pumpAndSettle();
    expect(
      find.text("Ejemplo validado · No se guardaron datos"),
      findsOneWidget,
    );
  });
  testWidgets("late completion after unmount does not update disposed form", (
    tester,
  ) async {
    final c = TextEditingController(text: "Ejemplo");
    addTearDown(c.dispose);
    final result = Completer<bool>();
    await tester.pumpWidget(
      BPApp(
        home: BPForm(
          fields: [BPTextField(id: "a", label: "Nombre", controller: c)],
          onSubmit: (_) => result.future,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text("Guardar"));
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox.shrink());
    result.complete(true);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });
}
