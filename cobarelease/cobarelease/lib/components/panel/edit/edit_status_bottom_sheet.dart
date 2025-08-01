import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/paneldisplaydata.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class EditStatusBottomSheet extends StatefulWidget {
  final String? panelVendorName;
  final String? busbarVendorName;
  final String duration;
  final DateTime? startDate;
  final double progress;
  final TextEditingController? vendorBusbarNameController;
  final PanelDisplayData panelData;
  final Company currentCompany;
  final VoidCallback onSave;

  const EditStatusBottomSheet({
    super.key,
    this.panelVendorName,
    this.vendorBusbarNameController,
    required this.busbarVendorName,
    required this.panelData,
    required this.startDate,
    required this.progress,
    required this.duration,
    required this.currentCompany,
    required this.onSave,
  });

  @override
  State<EditStatusBottomSheet> createState() => _EditStatusBottomSheetState();
}

class _EditStatusBottomSheetState extends State<EditStatusBottomSheet> {
  late String? _selectedPccStatus;
  late String? _selectedMccStatus;
  late String? _selectedComponentStatus;
  late final TextEditingController _remarkController;
  bool _isLoading = false;

  late DateTime? _aoBusbarPcc;
  // late DateTime? _etaBusbarPcc;
  late DateTime? _aoBusbarMcc;
  // late DateTime? _etaBusbarMcc;
  // late DateTime? _aoComponent;
  // late DateTime? _etaComponent;

  bool get _isK5 => widget.currentCompany.role == AppRole.k5;
  bool get _isWHS => widget.currentCompany.role == AppRole.warehouse;

  final List<String> _busbarStatusOptions = [
    "On Progress",
    "Siap 100%",
    "Close",
    "Red Block",
  ];
  final List<String> _componentStatusOptions = ["Open", "On Progress", "Done"];

  @override
  void initState() {
    super.initState();
    _remarkController = TextEditingController(
      text: widget.panelData.busbarRemarks ?? '',
    );

    final panel = widget.panelData.panel;

    if (_isK5) {
      _selectedPccStatus = panel.statusBusbarPcc ?? "On Progress";
      _selectedMccStatus = panel.statusBusbarMcc ?? "On Progress";
      _aoBusbarPcc = panel.aoBusbarPcc;
      // _etaBusbarPcc = panel.etaBusbarPcc;
      _aoBusbarMcc = panel.aoBusbarMcc;
      // _etaBusbarMcc = panel.etaBusbarMcc;
    } else if (_isWHS) {
      _selectedComponentStatus = panel.statusComponent ?? "Open";
      // _aoComponent = panel.aoComponent;
      // _etaComponent = panel.etaComponent;
    }
  }

  @override
  void dispose() {
    _remarkController.dispose();
    super.dispose();
  }

  Future<void> _saveChanges() async {
    setState(() => _isLoading = true);
    try {
      final panelToUpdate = await DatabaseHelper.instance.getPanelByNoPp(
        widget.panelData.panel.noPp,
      );
      if (panelToUpdate != null) {
        if (_isK5) {
          panelToUpdate.statusBusbarPcc = _selectedPccStatus;
          panelToUpdate.statusBusbarMcc = _selectedMccStatus;
          panelToUpdate.aoBusbarPcc = _aoBusbarPcc;
          // panelToUpdate.etaBusbarPcc = _etaBusbarPcc;
          panelToUpdate.aoBusbarMcc = _aoBusbarMcc;
          // panelToUpdate.etaBusbarMcc = _etaBusbarMcc;

          await DatabaseHelper.instance.updatePanel(panelToUpdate);
          await DatabaseHelper.instance.upsertBusbarRemarkandVendor(
            panelNoPp: widget.panelData.panel.noPp,
            vendorId: widget.currentCompany.id,
            newRemark: _remarkController.text,
          );
        } else if (_isWHS) {
          panelToUpdate.statusComponent = _selectedComponentStatus;
          // panelToUpdate.aoComponent = _aoComponent;
          // panelToUpdate.etaComponent = _etaComponent;
          await DatabaseHelper.instance.updatePanel(panelToUpdate);
        }
      }
      if (mounted) {
        widget.onSave();
        Navigator.pop(context);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text("Gagal menyimpan: ${e.toString()}"),
            backgroundColor: Colors.red,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  String _getBusbarStatusImage(String status) {
    final lower = status.toLowerCase();
    if (lower == 'n/a' || lower.contains('on progress')) {
      return 'assets/images/new-yellow.png';
    } else if (lower.contains('close')) {
      return 'assets/images/done-green.png';
    } else if (lower.contains('siap 100%')) {
      return 'assets/images/done-blue.png';
    } else if (lower.contains('red block')) {
      return 'assets/images/on-block-red.png';
    }
    return 'assets/images/no-status-gray.png';
  }

  String _getComponentStatusImage(String status) {
    final lower = status.toLowerCase();
    if (lower == 'n/a' || lower.contains('open')) {
      return 'assets/images/no-status-gray.png';
    } else if (lower.contains('done')) {
      return 'assets/images/done-green.png';
    } else if (lower.contains('on progress')) {
      return 'assets/images/on-progress-blue.png';
    }
    return 'assets/images/no-status-gray.png';
  }

  Color _getProgressColor(double progress) {
    if (progress < 0.5) return AppColors.red;
    if (progress < 1.0) return AppColors.orange;
    return AppColors.schneiderGreen;
  }

  String _getProgressImage(double progress) {
    if (progress < 0.5) return 'assets/images/progress-bolt-red.png';
    if (progress < 1.0) return 'assets/images/progress-bolt-orange.png';
    return 'assets/images/progress-bolt-green.png';
  }

  @override
  Widget build(BuildContext context) {
    final bool isFuture =
        widget.startDate != null && widget.startDate!.isAfter(DateTime.now());
    final String durationLabel = isFuture ? "Mulai Dalam" : "Durasi Proses";
    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        20,
        16,
        20,
        MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              height: 5,
              width: 40,
              decoration: BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.circular(100),
              ),
            ),
          ),
          const SizedBox(height: 24),
          _buildStatusCard(durationLabel),
          if (_isK5) ...[const SizedBox(height: 16), _buildRemarkField()],
          const SizedBox(height: 32),
          _buildActionButtons(),
        ],
      ),
    );
  }

  Widget _buildStatusCard(String durationLabel) {
    final panel = widget.panelData.panel;
    final progress = (panel.percentProgress ?? 0) / 100.0;

    return Container(
      decoration: BoxDecoration(
        borderRadius: const BorderRadius.all(Radius.circular(8)),
        border: Border.all(width: 1, color: AppColors.grayLight),
      ),
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.all(12),
            decoration: const BoxDecoration(
              color: AppColors.white,
              borderRadius: BorderRadius.only(
                topLeft: Radius.circular(7),
                topRight: Radius.circular(7),
              ),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Row(
                  children: [
                    Image.asset(_getProgressImage(progress), height: 28),
                    const SizedBox(width: 8),
                    Container(
                      padding: const EdgeInsets.only(right: 8),
                      decoration: const BoxDecoration(
                        border: Border(
                          right: BorderSide(
                            color: AppColors.grayNeutral,
                            width: 1,
                          ),
                        ),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            widget.duration,
                            style: const TextStyle(
                              color: AppColors.black,
                              fontWeight: FontWeight.w400,
                              fontSize: 12,
                            ),
                          ),
                          Text(
                            durationLabel,
                            style: const TextStyle(
                              color: AppColors.gray,
                              fontWeight: FontWeight.w400,
                              fontSize: 10,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Row(
                      children: [
                        const Text(
                          "Panel",
                          style: TextStyle(
                            color: AppColors.gray,
                            fontWeight: FontWeight.w400,
                            fontSize: 10,
                          ),
                        ),
                        const SizedBox(width: 4),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 4),
                          decoration: BoxDecoration(
                            color: AppColors.grayLight,
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            widget.panelVendorName ?? "N/A",
                            style: const TextStyle(
                              color: AppColors.black,
                              fontWeight: FontWeight.w400,
                              fontSize: 10,
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        Container(
                          width: MediaQuery.of(context).size.width - 280,
                          height: 11,
                          decoration: BoxDecoration(
                            color: AppColors.gray.withOpacity(0.3),
                            borderRadius: BorderRadius.circular(20),
                          ),
                          child: FractionallySizedBox(
                            alignment: Alignment.centerLeft,
                            widthFactor: progress.clamp(0.0, 1.0),
                            child: Container(
                              decoration: BoxDecoration(
                                color: _getProgressColor(progress),
                                borderRadius: BorderRadius.circular(20),
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          "${(progress * 100).toStringAsFixed(0)}%",
                          style: const TextStyle(
                            color: AppColors.black,
                            fontWeight: FontWeight.w500,
                            fontSize: 12,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ],
            ),
          ),
          Container(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  panel.noPanel,
                  style: const TextStyle(
                    color: AppColors.black,
                    fontWeight: FontWeight.w600,
                    fontSize: 16,
                  ),
                ),
                const SizedBox(height: 12),
                _buildVendornameField(),
                const SizedBox(height: 20),
                if (_isK5) ...[
                  _buildStatusOptionsList(
                    title: "Status Busbar PCC",
                    selectedValue: _selectedPccStatus,
                    onChanged: (newValue) {
                      setState(() => _selectedPccStatus = newValue);
                    },
                  ),
                  _buildAODatePicker(
                    "AO Busbar PCC",
                    _aoBusbarPcc,
                    (date) => setState(() => _aoBusbarPcc = date),
                  ),
                  // _buildETADatePicker(
                  //   "ETA Busbar PCC",
                  //   _etaBusbarPcc,
                  //   (date) => setState(() => _etaBusbarPcc = date),
                  // ),
                  const SizedBox(height: 20),
                  _buildStatusOptionsList(
                    title: "Status Busbar MCC",
                    selectedValue: _selectedMccStatus,
                    onChanged: (newValue) {
                      setState(() => _selectedMccStatus = newValue);
                    },
                  ),
                  _buildAODatePicker(
                    "AO Busbar MCC",
                    _aoBusbarMcc,
                    (date) => setState(() => _aoBusbarMcc = date),
                  ),
                  // _buildETADatePicker(
                  //   "ETA Busbar MCC",
                  //   _etaBusbarMcc,
                  //   (date) => setState(() => _etaBusbarMcc = date),
                  // ),
                ] else if (_isWHS) ...[
                  _buildStatusOptionsList(
                    title: "Status Picking Component",
                    selectedValue: _selectedComponentStatus,
                    onChanged: (newValue) {
                      setState(() => _selectedComponentStatus = newValue);
                    },
                  ),
                  // _buildAODatePicker(
                  //   "AO Component",
                  //   _aoComponent,
                  //   (date) => setState(() => _aoComponent = date),
                  // ),
                  // _buildETADatePicker(
                  //   "ETA Component",
                  //   _etaComponent,
                  //   (date) => setState(() => _etaComponent = date),
                  // ),
                ],
                const Divider(height: 24, color: AppColors.grayLight),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text(
                      "No. PP",
                      style: TextStyle(fontSize: 12, color: AppColors.gray),
                    ),
                    Text(panel.noPp, style: const TextStyle(fontSize: 12)),
                  ],
                ),
                const SizedBox(height: 4),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text(
                      "No. WBS",
                      style: TextStyle(fontSize: 12, color: AppColors.gray),
                    ),
                    Text(panel.noWbs, style: const TextStyle(fontSize: 12)),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStatusOptionsList({
    required String title,
    required String? selectedValue,
    required ValueChanged<String?> onChanged,
  }) {
    final options = _isK5 ? _busbarStatusOptions : _componentStatusOptions;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: const TextStyle(
            color: AppColors.black,
            fontWeight: FontWeight.w500,
            fontSize: 14,
          ),
        ),
        const SizedBox(height: 8),
        ...options.map(
          (status) => _buildStatusOptionRow(
            status: status,
            groupValue: selectedValue,
            onChanged: onChanged,
          ),
        ),
      ],
    );
  }

  Widget _buildStatusOptionRow({
    required String status,
    required String? groupValue,
    required ValueChanged<String?> onChanged,
  }) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: () => onChanged(status),
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 6.0),
          child: Row(
            children: [
              Text(
                status,
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w400,
                ),
              ),
              const SizedBox(width: 4),
              Image.asset(
                _isK5
                    ? _getBusbarStatusImage(status)
                    : _getComponentStatusImage(status),
                height: 12,
              ),
              const Spacer(),
              SizedBox(
                height: 24,
                width: 24,
                child: Radio<String>(
                  value: status,
                  groupValue: groupValue,
                  onChanged: (value) => onChanged(value),
                  activeColor: AppColors.schneiderGreen,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildAODatePicker(
    String label,
    DateTime? selectedDate,
    ValueChanged<DateTime> onDateChanged,
  ) {
    return _buildDatePicker(
      label,
      selectedDate,
      onDateChanged,
      Icons.assignment_turned_in_outlined,
    );
  }

  Widget _buildETADatePicker(
    String label,
    DateTime? selectedDate,
    ValueChanged<DateTime> onDateChanged,
  ) {
    return _buildDatePicker(
      label,
      selectedDate,
      onDateChanged,
      Icons.local_shipping_outlined,
    );
  }

  Widget _buildDatePicker(
    String label,
    DateTime? selectedDate,
    ValueChanged<DateTime> onDateChanged,
    IconData icon,
  ) {
    Future<void> pickDate() async {
      final date = await showDatePicker(
        context: context,
        initialDate: selectedDate ?? DateTime.now(),
        firstDate: DateTime(2000),
        lastDate: DateTime(2101),
        builder: (context, child) {
          return Theme(
            data: ThemeData.light().copyWith(
              colorScheme: const ColorScheme.light(
                primary: AppColors.schneiderGreen,
                onPrimary: Colors.white,
                onSurface: AppColors.black,
              ),
            ),
            child: child!,
          );
        },
      );
      if (date != null) {
        onDateChanged(date);
      }
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const SizedBox(height: 8),
        InkWell(
          onTap: pickDate,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: AppColors.grayLight),
            ),
            child: Row(
              children: [
                Icon(icon, size: 20, color: AppColors.gray),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    label,
                    style: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w400,
                    ),
                  ),
                ),
                Text(
                  selectedDate != null
                      ? DateFormat('d MMM yyyy').format(selectedDate)
                      : 'Pilih Tanggal',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w400,
                    color: selectedDate != null
                        ? AppColors.black
                        : AppColors.gray,
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildVendornameField() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          _isK5 ? "Vendor Busbar" : "Vendor Component",
          style: const TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w500,
            color: AppColors.black,
          ),
        ),
        const SizedBox(height: 8),
        TextFormField(
          initialValue: widget.currentCompany.name,
          maxLines: 1,
          style: const TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w300,
            color: AppColors.black,
          ),
          enabled: false,
          decoration: InputDecoration(
            fillColor: AppColors.grayLight,
            filled: true,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 16,
              vertical: 12,
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            disabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildRemarkField() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Remark",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
        ),
        const SizedBox(height: 8),
        TextFormField(
          controller: _remarkController,
          cursorColor: AppColors.schneiderGreen,
          maxLines: 3,
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
          decoration: InputDecoration(
            hintText: 'Masukkan remark...',
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 16,
              vertical: 12,
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.schneiderGreen),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildActionButtons() {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: () => Navigator.pop(context),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              side: const BorderSide(color: AppColors.schneiderGreen),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            child: const Text(
              "Batal",
              style: TextStyle(
                color: AppColors.schneiderGreen,
                fontWeight: FontWeight.w400,
              ),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: _saveChanges,
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              backgroundColor: AppColors.schneiderGreen,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadiusGeometry.all(Radius.circular(6)),
              ),
            ),
            child: _isLoading
                ? const SizedBox(
                    height: 18,
                    width: 18,
                    child: CircularProgressIndicator(
                      color: Colors.white,
                      strokeWidth: 2,
                    ),
                  )
                : const Text(
                    "Simpan",
                    style: TextStyle(fontWeight: FontWeight.w400),
                  ),
          ),
        ),
      ],
    );
  }
}
