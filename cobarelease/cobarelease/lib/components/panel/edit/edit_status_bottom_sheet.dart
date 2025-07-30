import 'package:flutter/material.dart';
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
  late String? _selectedStatus;
  late final TextEditingController _remarkController;
  bool _isLoading = false;

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

    if (_isK5) {
      _selectedStatus = widget.panelData.panel.statusBusbar ?? "On Progress";
    } else if (_isWHS) {
      _selectedStatus = widget.panelData.panel.statusComponent ?? "Open";
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
          final bool wasBusbarUnassigned =
              widget.panelData.busbarVendorIds.isEmpty;

          panelToUpdate.statusBusbar = wasBusbarUnassigned
              ? 'On Progress'
              : _selectedStatus;

          await DatabaseHelper.instance.updatePanel(panelToUpdate);
          await DatabaseHelper.instance.upsertBusbarRemarkandVendor(
            panelNoPp: widget.panelData.panel.noPp,
            vendorId: widget.currentCompany.id,
            newRemark: _remarkController.text,
          );
        } else if (_isWHS) {
          panelToUpdate.statusComponent = _selectedStatus;
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
          if (_isK5) ...[const SizedBox(height: 16), _buildVendornameField()],
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
                            style: TextStyle(
                              color: AppColors.black,
                              fontWeight: FontWeight.w400,
                              fontSize: 12,
                            ),
                          ),
                          Text(
                            durationLabel,
                            style: TextStyle(
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
                        Text(
                          "Panel",
                          style: const TextStyle(
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
                    SizedBox(height: 4),
                    Row(
                      children: [
                        Container(
                          width: MediaQuery.of(context).size.width - 244,
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
                        SizedBox(width: 8),
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
                _buildStatusOptionsList(),
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

  Widget _buildStatusOptionsList() {
    final options = _isK5 ? _busbarStatusOptions : _componentStatusOptions;
    final title = _isK5 ? "Status Busbar" : "Status Picking Component";

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              title,
              style: const TextStyle(
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
                _isK5
                    ? widget.currentCompany.name
                    : widget.busbarVendorName ?? "N/A",
                style: const TextStyle(
                  color: AppColors.black,
                  fontWeight: FontWeight.w400,
                  fontSize: 10,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        ...options.map((status) => _buildStatusOptionRow(status)),
      ],
    );
  }

  Widget _buildStatusOptionRow(String status) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: () {
          setState(() {
            _selectedStatus = status;
          });
        },
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 6.0),
          child: Row(
            children: [
              Text(
                status,
                style: const TextStyle(
                  fontSize: 14,
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
                child: Checkbox(
                  value: _selectedStatus == status,
                  onChanged: (value) {
                    setState(() {
                      _selectedStatus = status;
                    });
                  },
                  activeColor: AppColors.schneiderGreen,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildVendornameField() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Vendor Busbar",
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w500,
            color: AppColors.black,
          ),
        ),
        widget.panelData.busbarVendorIds.isEmpty
            ? Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    "Vendor akan di-assign otomatis jika menyimpan status busbar",
                    style: TextStyle(
                      fontSize: 12,
                      color: AppColors.gray,
                      fontWeight: FontWeight.w300,
                    ),
                  ),
                  SizedBox(height: 8),
                  Row(
                    children: [
                      Expanded(
                        child: TextFormField(
                          initialValue: 'N/A',
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
                              borderSide: const BorderSide(
                                color: AppColors.grayLight,
                              ),
                            ),
                            disabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                              borderSide: const BorderSide(
                                color: AppColors.grayLight,
                              ),
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Icon(
                        Icons.arrow_circle_right,
                        size: 24,
                        color: AppColors.schneiderGreen,
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: TextFormField(
                          controller: TextEditingController(
                            text: widget.currentCompany.name,
                          ),
                          maxLines: 1,
                          style: const TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w400,
                            color: AppColors.black,
                          ),
                          enabled: false,
                          decoration: InputDecoration(
                            fillColor: AppColors.schneiderGreen.withOpacity(
                              0.1,
                            ),
                            filled: true,
                            contentPadding: const EdgeInsets.symmetric(
                              horizontal: 16,
                              vertical: 12,
                            ),
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                              borderSide: const BorderSide(
                                color: AppColors.schneiderGreen,
                              ),
                            ),
                            disabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                              borderSide: const BorderSide(
                                color: AppColors.schneiderGreen,
                              ),
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ],
              )
            : Column(
                children: [
                  SizedBox(height: 8),
                  TextFormField(
                    initialValue: widget.busbarVendorName,
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
                      hintText: 'Masukkan vendor busbar...',
                      hintStyle: const TextStyle(color: AppColors.gray),
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 12,
                      ),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: const BorderSide(
                          color: AppColors.grayLight,
                        ),
                      ),
                      disabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: const BorderSide(
                          color: AppColors.grayLight,
                        ),
                      ),
                    ),
                  ),
                ],
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
        (!widget.panelData.busbarVendorIds.contains(widget.currentCompany.id) &&
                widget.panelData.busbarVendorIds.isNotEmpty)
            ? TextFormField(
                initialValue: (_remarkController.text.isEmpty
                    ? "Tidak ada catatan"
                    : _remarkController.text),
                maxLines: 3,
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
              )
            : TextFormField(
                controller: _remarkController,
                cursorColor: AppColors.schneiderGreen,
                maxLines: 3,
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w300,
                ),
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
                    borderSide: const BorderSide(
                      color: AppColors.schneiderGreen,
                    ),
                  ),
                ),
              ),
      ],
    );
  }

  Widget _buildActionButtons() {
    return (!widget.panelData.busbarVendorIds.contains(
              widget.currentCompany.id,
            ) &&
            widget.panelData.busbarVendorIds.isNotEmpty)
        ? InkWell(
            onTap: () => Navigator.pop(context),
            child: Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(vertical: 16),
              decoration: BoxDecoration(
                border: Border.all(color: AppColors.schneiderGreen),
                borderRadius: BorderRadius.circular(6),
              ),
              child: const Center(
                child: Text(
                  "Tutup",
                  style: TextStyle(
                    color: AppColors.schneiderGreen,
                    fontSize: 12,
                  ),
                ),
              ),
            ),
          )
        : Row(
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
                      fontSize: 12,
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
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
                  child: _isLoading
                      ? const SizedBox(
                          height: 16,
                          width: 16,
                          child: CircularProgressIndicator(
                            color: Colors.white,
                            strokeWidth: 2,
                          ),
                        )
                      : const Text(
                          "Simpan",
                          style: TextStyle(color: Colors.white, fontSize: 12),
                        ),
                ),
              ),
            ],
          );
  }
}
