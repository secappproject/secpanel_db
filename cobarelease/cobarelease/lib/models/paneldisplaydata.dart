import 'package:secpanel/models/panels.dart';

class PanelDisplayData {
  final Panel panel;
  final String panelVendorName;
  final String busbarVendorNames;
  final List<String> busbarVendorIds;
  final String componentVendorNames;
  final List<String> componentVendorIds;
  final String paletVendorNames;
  final List<String> paletVendorIds;
  final String corepartVendorNames;
  final List<String> corepartVendorIds;
  final String? busbarRemarks;

  PanelDisplayData({
    required this.panel,
    required this.panelVendorName,
    required this.busbarVendorNames,
    required this.busbarVendorIds,
    required this.componentVendorNames,
    required this.componentVendorIds,
    required this.paletVendorNames,
    required this.paletVendorIds,
    required this.corepartVendorNames,
    required this.corepartVendorIds,
    this.busbarRemarks,
  });
}
