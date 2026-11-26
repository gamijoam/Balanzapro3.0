import "dart:convert";

import "package:flutter/services.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";
import "package:balanzapro_alpha/main.dart";

void main() {
  setUpAll(() async {
    final manifest =
        jsonDecode(await rootBundle.loadString("FontManifest.json")) as List;
    for (final entry in manifest) {
      final family = entry["family"] as String;
      final loader = FontLoader(family);
      for (final font in (entry["fonts"] as List)) {
        loader.addFont(
          rootBundle.load(Uri.decodeComponent(font["asset"] as String)),
        );
      }
      await loader.load();
    }
  });
  for (final width in [360.0, 1440.0]) {
    testWidgets("capture actual Flutter rendering $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final dispatcher = tester.binding.platformDispatcher;
      dispatcher.platformBrightnessTestValue = Brightness.light;
      addTearDown(dispatcher.clearPlatformBrightnessTestValue);
      await tester.pumpWidget(
        const RepaintBoundary(key: ValueKey("capture"), child: BalanzaProApp()),
      );
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      await expectLater(
        find.byKey(const ValueKey("capture")),
        matchesGoldenFile("visual-evidence/products-${width.toInt()}.png"),
      );
    });
  }
}
