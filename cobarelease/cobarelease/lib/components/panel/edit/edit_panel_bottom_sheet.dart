import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/paneldisplaydata.dart';
import 'package:secpanel/models/panels.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/models/busbar.dart';
import 'package:secpanel/models/component.dart';
import 'package:secpanel/models/palet.dart';
import 'package:secpanel/models/corepart.dart';
import 'package:secpanel/theme/colors.dart';

class EditPanelBottomSheet extends StatefulWidget {
  final PanelDisplayData panelData;
  final Company currentCompany;
  final List<Company> k3Vendors;
  final Function(Panel) onSave;
  final VoidCallback onDelete;

  const EditPanelBottomSheet({
    super.key,
    required this.panelData,
    required this.currentCompany,
    required this.k3Vendors,
    required this.onSave,
    required this.onDelete,
  });

  @override
  State<EditPanelBottomSheet> createState() => _EditPanelBottomSheetState();
}

class _EditPanelBottomSheetState extends State<EditPanelBottomSheet> {
  final _formKey = GlobalKey<FormState>();

  late final TextEditingController _noPanelController;
  late final TextEditingController _noWbsController;
  late final TextEditingController _noPpController;
  late final TextEditingController _progressController;

  late Panel _panel;
  late DateTime _selectedDate;
  late DateTime? _selectedTargetDeliveryDate;
  late String? _selectedK3VendorId;
  late bool _isSent;
  bool _canMarkAsSent = false;

  bool _isLoading = false;
  bool _isSuccess = false;

  bool get _isAdmin => widget.currentCompany.role == AppRole.admin;
  bool get _isK3 => widget.currentCompany.role == AppRole.k3;

  List<Company> _k5Vendors = [];
  List<Company> _whsVendors = [];
  String? _selectedBusbarVendorId;
  String? _selectedComponentVendorId;
  String? _selectedPaletVendorId;
  String? _selectedCorepartVendorId;

  String? _selectedBusbarPccStatus;
  String? _selectedBusbarMccStatus;
  String? _selectedComponentStatus;
  String? _selectedPaletStatus;
  String? _selectedCorepartStatus;

  DateTime? _aoBusbarPcc;
  DateTime? _aoBusbarMcc;

  final List<String> busbarStatusOptions = [
    "On Progress",
    "Siap 100%",
    "Close",
    "Red Block",
  ];
  final List<String> componentStatusOptions = ["Open", "On Progress", "Done"];
  final List<String> paletCorepartStatusOptions = ["Open", "Close"];

  @override
  void initState() {
    super.initState();
    _panel = Panel.fromMap(widget.panelData.panel.toMap());

    _noPanelController = TextEditingController(text: _panel.noPanel);
    _noWbsController = TextEditingController(text: _panel.noWbs);
    _noPpController = TextEditingController(text: _panel.noPp);
    _progressController = TextEditingController(
      text: _panel.percentProgress?.toInt().toString() ?? '0',
    );
    _selectedDate = _panel.startDate ?? DateTime.now();
    _selectedTargetDeliveryDate = _panel.targetDelivery;
    _selectedK3VendorId = _panel.vendorId;
    _isSent = _panel.isClosed;

    _selectedBusbarPccStatus = _panel.statusBusbarPcc;
    _selectedBusbarMccStatus = _panel.statusBusbarMcc;
    _selectedComponentStatus = _panel.statusComponent;
    _selectedPaletStatus = _panel.statusPalet;
    _selectedCorepartStatus = _panel.statusCorepart;

    _aoBusbarPcc = _panel.aoBusbarPcc;
    _aoBusbarMcc = _panel.aoBusbarMcc;

    WidgetsBinding.instance.addPostFrameCallback((_) => _updateCanMarkAsSent());
    _progressController.addListener(_updateCanMarkAsSent);

    _loadVendors();
  }

  Future<void> _loadVendors() async {
    final k5 = await DatabaseHelper.instance.getK5Vendors();
    final whs = await DatabaseHelper.instance.getWHSVendors();
    if (mounted) {
      setState(() {
        _k5Vendors = k5;
        _whsVendors = whs;
        _selectedBusbarVendorId = widget.panelData.busbarVendorIds.isNotEmpty
            ? widget.panelData.busbarVendorIds.first
            : null;
        _selectedComponentVendorId =
            widget.panelData.componentVendorIds.isNotEmpty
            ? widget.panelData.componentVendorIds.first
            : null;
        _selectedPaletVendorId = widget.panelData.paletVendorIds.isNotEmpty
            ? widget.panelData.paletVendorIds.first
            : null;
        _selectedCorepartVendorId =
            widget.panelData.corepartVendorIds.isNotEmpty
            ? widget.panelData.corepartVendorIds.first
            : null;
      });
    }
  }

  @override
  void dispose() {
    _noPanelController.dispose();
    _noWbsController.dispose();
    _noPpController.dispose();
    _progressController.removeListener(_updateCanMarkAsSent);
    _progressController.dispose();
    super.dispose();
  }

  void _updateCanMarkAsSent() {
    final progress = int.tryParse(_progressController.text) ?? 0;
    final paletReady = _selectedPaletStatus == 'Close';
    final corepartReady = _selectedCorepartStatus == 'Close';
    final busbarMccReady = _selectedBusbarMccStatus == 'Close';
    final allConditionsMet =
        progress == 100 && paletReady && corepartReady && busbarMccReady;

    if (mounted && _canMarkAsSent != allConditionsMet) {
      setState(() {
        _canMarkAsSent = allConditionsMet;
        if (!_canMarkAsSent) _isSent = false;
      });
    }
  }

  Future<void> _saveChanges() async {
    if (_isLoading || _isSuccess) return;
    if (_formKey.currentState!.validate()) {
      setState(() => _isLoading = true);

      _panel.noPanel = _noPanelController.text.trim();
      _panel.noWbs = _noWbsController.text.trim();
      _panel.noPp = _noPpController.text.trim();
      _panel.percentProgress =
          double.tryParse(_progressController.text.trim()) ?? 0.0;
      _panel.startDate = _selectedDate;
      _panel.targetDelivery = _selectedTargetDeliveryDate;
      _panel.vendorId = _selectedK3VendorId;
      _panel.isClosed = _isSent;

      if (_isAdmin || _isK3) {
        _panel.statusBusbarPcc = _selectedBusbarPccStatus;
        _panel.statusBusbarMcc = _selectedBusbarMccStatus;
        _panel.statusComponent = _selectedComponentStatus;
        _panel.statusPalet = _selectedPaletStatus;
        _panel.statusCorepart = _selectedCorepartStatus;
        _panel.aoBusbarPcc = _aoBusbarPcc;
        _panel.aoBusbarMcc = _aoBusbarMcc;
      }

      if (_isSent && _panel.closedDate == null) {
        _panel.closedDate = DateTime.now();
      } else if (!_isSent) {
        _panel.closedDate = null;
      }

      try {
        await DatabaseHelper.instance.updatePanel(_panel);
        if (_isAdmin) {
          if (_selectedBusbarVendorId != null)
            await DatabaseHelper.instance.upsertBusbar(
              Busbar(panelNoPp: _panel.noPp, vendor: _selectedBusbarVendorId!),
            );
          if (_selectedComponentVendorId != null)
            await DatabaseHelper.instance.upsertComponent(
              Component(
                panelNoPp: _panel.noPp,
                vendor: _selectedComponentVendorId!,
              ),
            );
        }
        if (_isAdmin || _isK3) {
          if (_selectedPaletVendorId != null)
            await DatabaseHelper.instance.upsertPalet(
              Palet(panelNoPp: _panel.noPp, vendor: _selectedPaletVendorId!),
            );
          if (_selectedCorepartVendorId != null)
            await DatabaseHelper.instance.upsertCorepart(
              Corepart(
                panelNoPp: _panel.noPp,
                vendor: _selectedCorepartVendorId!,
              ),
            );
        }

        setState(() {
          _isLoading = false;
          _isSuccess = true;
        });
        await Future.delayed(const Duration(milliseconds: 1500));

        if (mounted) {
          widget.onSave(_panel);
          Navigator.pop(context);
        }
      } catch (e) {
        if (mounted) {
          setState(() => _isLoading = false);
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text('Gagal menyimpan: ${e.toString()}'),
              backgroundColor: Colors.red,
            ),
          );
        }
      }
    }
  }

  void _showDeleteConfirmation() {
    // Kode ini tidak berubah
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: SingleChildScrollView(
        child: Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 16,
            bottom: MediaQuery.of(context).viewInsets.bottom + 16,
          ),
          child: Form(
            key: _formKey,
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
                Row(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(
                      child: Text(
                        "Edit Panel ${_panel.noPanel}",
                        style: const TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ),
                    if (_isAdmin)
                      IconButton(
                        icon: const Icon(
                          Icons.delete_outline,
                          color: AppColors.red,
                        ),
                        onPressed: _showDeleteConfirmation,
                      ),
                  ],
                ),
                SizedBox(height: 12),
                Container(
                  width: MediaQuery.of(context).size.width,
                  padding: EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    border: BoxBorder.all(color: AppColors.grayLight, width: 1),
                    borderRadius: BorderRadius.all(Radius.circular(12)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _buildSectionHeader("Panel"),
                      if (_isAdmin || _isK3) ...[
                        _buildMarkAsSent(),
                        const SizedBox(height: 16),
                      ],
                      _buildTextField(
                        controller: _noPanelController,
                        label: "No. Panel",
                      ),
                      const SizedBox(height: 16),
                      _buildTextField(
                        controller: _noWbsController,
                        label: "No. WBS",
                      ),
                      const SizedBox(height: 16),
                      _buildTextField(
                        controller: _noPpController,
                        label: "No. PP",
                      ),
                      const SizedBox(height: 16),
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            flex: 1,
                            child: _buildTextField(
                              controller: _progressController,
                              label: "Progress",
                              isNumber: true,
                              suffixText: "%",
                            ),
                          ),
                          const SizedBox(width: 16),
                          Expanded(flex: 2, child: _buildDateTimePicker()),
                        ],
                      ),
                      const SizedBox(height: 16),
                      _buildTargetDeliveryPicker(),
                      const SizedBox(height: 16),
                      if (_isAdmin)
                        _buildAdminVendorPicker()
                      else if (_isK3)
                        _buildK3VendorDisplay(),
                    ],
                  ),
                ),

                if (_isAdmin) ...[
                  SizedBox(height: 12),
                  Container(
                    width: MediaQuery.of(context).size.width,
                    padding: EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      border: BoxBorder.all(
                        color: AppColors.grayLight,
                        width: 1,
                      ),
                      borderRadius: BorderRadius.all(Radius.circular(12)),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildSectionHeader("Busbar"),
                        _buildSelectorSection(
                          label: "Vendor Busbar (K5)",
                          options: Map.fromEntries(
                            _k5Vendors.map((v) => MapEntry(v.id, v.name)),
                          ),
                          selectedValue: _selectedBusbarVendorId,
                          onTap: (val) =>
                              setState(() => _selectedBusbarVendorId = val),
                        ),
                        const SizedBox(height: 16),
                        _buildSelectorSection(
                          label: "Status Busbar PCC",
                          options: Map.fromEntries(
                            busbarStatusOptions.map((s) => MapEntry(s, s)),
                          ),
                          selectedValue: _selectedBusbarPccStatus,
                          onTap: (val) => setState(() {
                            _selectedBusbarPccStatus = val;
                            _updateCanMarkAsSent();
                          }),
                          isEnabled: _selectedBusbarVendorId != null,
                        ),
                        const SizedBox(height: 16),
                        _buildDatePickerField(
                          label: "Acknowledgement Order PCC",
                          selectedDate: _aoBusbarPcc,
                          onDateChanged: (date) =>
                              setState(() => _aoBusbarPcc = date),
                          icon: Icons.assignment_turned_in_outlined,
                        ),
                        const SizedBox(height: 16),
                        _buildSelectorSection(
                          label: "Status Busbar MCC",
                          options: Map.fromEntries(
                            busbarStatusOptions.map((s) => MapEntry(s, s)),
                          ),
                          selectedValue: _selectedBusbarMccStatus,
                          onTap: (val) => setState(() {
                            _selectedBusbarMccStatus = val;
                            _updateCanMarkAsSent();
                          }),
                          isEnabled: _selectedBusbarVendorId != null,
                        ),
                        const SizedBox(height: 16),
                        _buildDatePickerField(
                          label: "Acknowledgement Order MCC",
                          selectedDate: _aoBusbarMcc,
                          onDateChanged: (date) =>
                              setState(() => _aoBusbarMcc = date),
                          icon: Icons.assignment_turned_in_outlined,
                        ),
                      ],
                    ),
                  ),
                  SizedBox(height: 12),
                  Container(
                    width: MediaQuery.of(context).size.width,
                    padding: EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      border: BoxBorder.all(
                        color: AppColors.grayLight,
                        width: 1,
                      ),
                      borderRadius: BorderRadius.all(Radius.circular(12)),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildSectionHeader("Komponen"),
                        _buildSelectorSection(
                          label: "Status Komponen",
                          options: Map.fromEntries(
                            componentStatusOptions.map((s) => MapEntry(s, s)),
                          ),
                          selectedValue: _selectedComponentStatus,
                          onTap: (val) => setState(() {
                            _selectedComponentStatus = val;
                            _updateCanMarkAsSent();
                          }),
                          isEnabled: _selectedComponentVendorId != null,
                        ),
                      ],
                    ),
                  ),
                ],
                SizedBox(height: 12),
                Container(
                  width: MediaQuery.of(context).size.width,
                  padding: EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    border: BoxBorder.all(color: AppColors.grayLight, width: 1),
                    borderRadius: BorderRadius.all(Radius.circular(12)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _buildSectionHeader("Palet"),
                      _buildSelectorSection(
                        label: "Status Palet",
                        options: Map.fromEntries(
                          paletCorepartStatusOptions.map((s) => MapEntry(s, s)),
                        ),
                        selectedValue: _selectedPaletStatus,
                        onTap: (val) => setState(() {
                          _selectedPaletStatus = val;
                          _updateCanMarkAsSent();
                        }),
                        isEnabled: _selectedPaletVendorId != null,
                      ),
                    ],
                  ),
                ),
                SizedBox(height: 12),
                Container(
                  width: MediaQuery.of(context).size.width,
                  padding: EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    border: BoxBorder.all(color: AppColors.grayLight, width: 1),
                    borderRadius: BorderRadius.all(Radius.circular(12)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _buildSectionHeader("Corepart"),
                      _buildSelectorSection(
                        label: "Status Corepart",
                        options: Map.fromEntries(
                          paletCorepartStatusOptions.map((s) => MapEntry(s, s)),
                        ),
                        selectedValue: _selectedCorepartStatus,
                        onTap: (val) => setState(() {
                          _selectedCorepartStatus = val;
                          _updateCanMarkAsSent();
                        }),
                        isEnabled: _selectedCorepartVendorId != null,
                      ),
                    ],
                  ),
                ),
                SizedBox(height: 12),
                _buildActionButtons(),
              ],
            ),
          ),
        ),
      ),
    );
  }

  // --- HELPER WIDGET BUILDERS ---

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16.0),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w500,
          color: AppColors.black,
        ),
      ),
    );
  }

  Widget _buildSectionDivider() {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 24.0),
      child: Divider(height: 1, color: AppColors.grayLight),
    );
  }

  Widget _buildTextField({
    required TextEditingController controller,
    required String label,
    bool isNumber = false,
    String? suffixText,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        TextFormField(
          cursorColor: AppColors.schneiderGreen,
          controller: controller,
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
          keyboardType: isNumber ? TextInputType.number : TextInputType.text,
          validator: (value) {
            if (isNumber) {
              if (value == null || value.isEmpty) return '0-100';
              final progress = int.tryParse(value);
              if (progress == null || progress < 0 || progress > 100)
                return '0-100';
            }
            if (!isNumber && (value == null || value.isEmpty))
              return 'Field ini tidak boleh kosong';
            return null;
          },
          decoration: InputDecoration(
            suffixText: suffixText,
            hintText: 'Masukkan $label',
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 16,
              vertical: 12,
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.grayLight),
            ),
            enabledBorder: OutlineInputBorder(
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

  // Method helper lainnya (_buildSelectorSection, _buildMarkAsSent, dll.) tidak berubah
  // dan bisa diletakkan di sini...
  Widget _buildSelectorSection({
    required String label,
    required Map<String, String> options,
    required String? selectedValue,
    required ValueChanged<String?> onTap,
    bool isEnabled = true,
  }) {
    // ... (kode _buildSelectorSection tidak berubah)
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w400,
            color: isEnabled ? AppColors.black : AppColors.gray,
          ),
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 12,
          children: options.entries.map((entry) {
            return _buildOptionButton(
              label: entry.value,
              selected: selectedValue == entry.key,
              onTap: isEnabled ? () => onTap(entry.key) : null,
            );
          }).toList(),
        ),
      ],
    );
  }

  Widget _buildMarkAsSent() {
    // ... (kode _buildMarkAsSent tidak berubah)
    final Color borderColor = _isSent
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    final Color bgColor = _isSent
        ? AppColors.schneiderGreen.withOpacity(0.08)
        : AppColors.white;
    final Color textColor = _canMarkAsSent ? AppColors.black : AppColors.gray;

    return InkWell(
      onTap: _canMarkAsSent ? () => setState(() => _isSent = !_isSent) : null,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: bgColor,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              Icons.local_shipping_outlined,
              color: _canMarkAsSent ? AppColors.schneiderGreen : AppColors.gray,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    "Tandai Sudah Dikirim",
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      color: textColor,
                    ),
                  ),
                  Text(
                    "Syarat: Progres 100%, Palet, Corepart & Busbar MCC Close.",
                    style: TextStyle(
                      fontSize: 10,
                      color: textColor,
                      fontWeight: FontWeight.w300,
                    ),
                  ),
                ],
              ),
            ),
            Checkbox(
              value: _isSent,
              onChanged: _canMarkAsSent
                  ? (bool? value) => setState(() => _isSent = value ?? false)
                  : null,
              activeColor: AppColors.schneiderGreen,
              checkColor: Colors.white,
              visualDensity: VisualDensity.compact,
              side: BorderSide(
                color: _canMarkAsSent ? AppColors.gray : AppColors.grayLight,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDateTimePicker() {
    // ... (kode _buildDateTimePicker tidak berubah)
    Future<void> pickDateTime() async {
      final date = await showDatePicker(
        context: context,
        initialDate: _selectedDate,
        firstDate: DateTime(2000),
        lastDate: DateTime(2101),
        initialEntryMode: DatePickerEntryMode.calendarOnly,
        builder: (context, child) => Theme(
          data: ThemeData.light().copyWith(
            colorScheme: const ColorScheme.light(
              primary: AppColors.schneiderGreen,
              onPrimary: Colors.white,
              onSurface: AppColors.black,
            ),
          ),
          child: child!,
        ),
      );
      if (date == null) return;
      final time = await showTimePicker(
        context: context,
        initialTime: TimeOfDay.fromDateTime(_selectedDate),
      );
      if (time == null) return;
      setState(
        () => _selectedDate = DateTime(
          date.year,
          date.month,
          date.day,
          time.hour,
          time.minute,
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Waktu Mulai Pengerjaan",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        InkWell(
          onTap: pickDateTime,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: AppColors.grayLight),
            ),
            child: Row(
              children: [
                const Icon(
                  Icons.calendar_today_outlined,
                  size: 20,
                  color: AppColors.gray,
                ),
                const SizedBox(width: 8),
                Text(
                  DateFormat('d MMM yyyy HH:mm').format(_selectedDate),
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w300,
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildTargetDeliveryPicker() {
    return _buildDatePickerField(
      label: 'Target Delivery',
      selectedDate: _selectedTargetDeliveryDate,
      onDateChanged: (date) =>
          setState(() => _selectedTargetDeliveryDate = date),
      icon: Icons.flag_outlined,
    );
  }

  Widget _buildDatePickerField({
    required String label,
    required DateTime? selectedDate,
    required ValueChanged<DateTime> onDateChanged,
    required IconData icon,
  }) {
    // ... (kode _buildDatePickerField tidak berubah)
    Future<void> pickDate() async {
      final date = await showDatePicker(
        context: context,
        initialDate: selectedDate ?? DateTime.now(),
        firstDate: DateTime(2000),
        lastDate: DateTime(2101),
        builder: (context, child) => Theme(
          data: ThemeData.light().copyWith(
            colorScheme: const ColorScheme.light(
              primary: AppColors.schneiderGreen,
              onPrimary: Colors.white,
              onSurface: AppColors.black,
            ),
          ),
          child: child!,
        ),
      );
      if (date != null) onDateChanged(date);
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        InkWell(
          onTap: pickDate,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            width: double.infinity,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: AppColors.grayLight),
            ),
            child: Row(
              children: [
                Icon(icon, size: 20, color: AppColors.gray),
                const SizedBox(width: 8),
                Text(
                  selectedDate != null
                      ? DateFormat('d MMM yyyy').format(selectedDate)
                      : 'Pilih Tanggal',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w300,
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

  Widget _buildAdminVendorPicker() {
    return _buildSelectorSection(
      label: "Vendor Panel (K3)",
      options: Map.fromEntries(
        widget.k3Vendors.map((v) => MapEntry(v.id, v.name)),
      ),
      selectedValue: _selectedK3VendorId,
      onTap: (val) => setState(() => _selectedK3VendorId = val),
    );
  }

  Widget _buildK3VendorDisplay() {
    // ... (kode _buildK3VendorDisplay tidak berubah)
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Vendor Panel",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: AppColors.grayLight.withOpacity(0.5),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: AppColors.grayLight),
          ),
          child: Text(
            widget.currentCompany.name,
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w300,
              color: AppColors.gray,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildOptionButton({
    required String label,
    required bool selected,
    required VoidCallback? onTap,
  }) {
    // ... (kode _buildOptionButton tidak berubah)
    final Color borderColor = selected
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: selected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : (onTap != null ? Colors.white : Colors.grey.shade100),
          border: Border.all(
            color: onTap != null ? borderColor : Colors.grey.shade300,
          ),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontWeight: FontWeight.w400,
            fontSize: 12,
            color: onTap != null ? AppColors.black : AppColors.gray,
          ),
        ),
      ),
    );
  }

  Widget _buildActionButtons() {
    // ... (kode _buildActionButtons tidak berubah)
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
              style: TextStyle(color: AppColors.schneiderGreen, fontSize: 12),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: _saveChanges,
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
              backgroundColor: _isSuccess
                  ? Colors.green
                  : AppColors.schneiderGreen,
              foregroundColor: Colors.white,
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
                : _isSuccess
                ? const Icon(Icons.check, size: 16)
                : const Text("Simpan", style: TextStyle(fontSize: 12)),
          ),
        ),
      ],
    );
  }
}
