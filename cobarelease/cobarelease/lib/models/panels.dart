import 'dart:convert';

class Panel {
  String noPp;
  // --- [PERBAIKAN] Ubah menjadi nullable String? ---
  String? noPanel;
  String? noWbs;
  String? project;
  double? percentProgress;
  DateTime? startDate;
  DateTime? targetDelivery;
  String? statusBusbarPcc;
  String? statusBusbarMcc;
  String? statusComponent;
  String? statusPalet;
  String? statusCorepart;
  DateTime? aoBusbarPcc;
  DateTime? aoBusbarMcc;
  String? createdBy;
  String? vendorId;
  bool isClosed;
  DateTime? closedDate;

  Panel({
    required this.noPp,
    // --- [PERBAIKAN] Hapus 'required' karena field boleh null ---
    this.noPanel,
    this.noWbs,
    this.project,
    this.percentProgress,
    this.startDate,
    this.targetDelivery,
    this.statusBusbarPcc,
    this.statusBusbarMcc,
    this.statusComponent,
    this.statusPalet,
    this.statusCorepart,
    this.aoBusbarPcc,
    this.aoBusbarMcc,
    this.createdBy,
    this.vendorId,
    this.isClosed = false,
    this.closedDate,
  });

  Map<String, dynamic> toMap() {
    return {
      'no_pp': noPp,
      'no_panel': noPanel,
      'no_wbs': noWbs,
      'project': project,
      'percent_progress': percentProgress,
      'start_date': startDate?.toIso8601String(),
      'target_delivery': targetDelivery?.toIso8601String(),
      'status_busbar_pcc': statusBusbarPcc,
      'status_busbar_mcc': statusBusbarMcc,
      'status_component': statusComponent,
      'status_palet': statusPalet,
      'status_corepart': statusCorepart,
      'ao_busbar_pcc': aoBusbarPcc?.toIso8601String(),
      'ao_busbar_mcc': aoBusbarMcc?.toIso8601String(),
      'created_by': createdBy,
      'vendor_id': vendorId,
      'is_closed': isClosed ? 1 : 0,
      'closed_date': closedDate?.toIso8601String(),
    };
  }

  factory Panel.fromMap(Map<String, dynamic> map) {
    return Panel(
      noPp: map['no_pp'] ?? '',
      noPanel: map['no_panel'],
      noWbs: map['no_wbs'],
      project: map['project'],
      percentProgress: map['percent_progress'],
      startDate: map['start_date'] != null
          ? DateTime.parse(map['start_date'])
          : null,
      targetDelivery: map['target_delivery'] != null
          ? DateTime.parse(map['target_delivery'])
          : null,
      statusBusbarPcc: map['status_busbar_pcc'],
      statusBusbarMcc: map['status_busbar_mcc'],
      statusComponent: map['status_component'],
      statusPalet: map['status_palet'],
      statusCorepart: map['status_corepart'],
      aoBusbarPcc: map['ao_busbar_pcc'] != null
          ? DateTime.parse(map['ao_busbar_pcc'])
          : null,
      aoBusbarMcc: map['ao_busbar_mcc'] != null
          ? DateTime.parse(map['ao_busbar_mcc'])
          : null,
      createdBy: map['created_by'],
      vendorId: map['vendor_id'],
      isClosed: map['is_closed'] == 1,
      closedDate: map['closed_date'] != null
          ? DateTime.parse(map['closed_date'])
          : null,
    );
  }

  String toJson() => json.encode(toMap());

  factory Panel.fromJson(String source) => Panel.fromMap(json.decode(source));

  Panel copyWith({
    String? noPp,
    String? noPanel,
    String? noWbs,
    String? project,
    double? percentProgress,
    DateTime? startDate,
    DateTime? targetDelivery,
    String? statusBusbarPcc,
    String? statusBusbarMcc,
    String? statusComponent,
    String? statusPalet,
    String? statusCorepart,
    DateTime? aoBusbarPcc,
    DateTime? aoBusbarMcc,
    String? createdBy,
    String? vendorId,
    bool? isClosed,
    DateTime? closedDate,
  }) {
    return Panel(
      noPp: noPp ?? this.noPp,
      noPanel: noPanel ?? this.noPanel,
      noWbs: noWbs ?? this.noWbs,
      project: project ?? this.project,
      percentProgress: percentProgress ?? this.percentProgress,
      startDate: startDate ?? this.startDate,
      targetDelivery: targetDelivery ?? this.targetDelivery,
      statusBusbarPcc: statusBusbarPcc ?? this.statusBusbarPcc,
      statusBusbarMcc: statusBusbarMcc ?? this.statusBusbarMcc,
      statusComponent: statusComponent ?? this.statusComponent,
      statusPalet: statusPalet ?? this.statusPalet,
      statusCorepart: statusCorepart ?? this.statusCorepart,
      aoBusbarPcc: aoBusbarPcc ?? this.aoBusbarPcc,
      aoBusbarMcc: aoBusbarMcc ?? this.aoBusbarMcc,
      createdBy: createdBy ?? this.createdBy,
      vendorId: vendorId ?? this.vendorId,
      isClosed: isClosed ?? this.isClosed,
      closedDate: closedDate ?? this.closedDate,
    );
  }
}
