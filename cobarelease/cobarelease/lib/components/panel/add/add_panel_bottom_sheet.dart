import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/component.dart';
import 'package:secpanel/models/corepart.dart';
import 'package:secpanel/models/palet.dart';
import 'package:secpanel/models/panels.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class AddPanelBottomSheet extends StatefulWidget {
  final Company currentCompany;
  final List<Company> k3Vendors;
  final Function(Panel) onPanelAdded;

  const AddPanelBottomSheet({
    super.key,
    required this.currentCompany,
    required this.k3Vendors,
    required this.onPanelAdded,
  });

  @override
  State<AddPanelBottomSheet> createState() => _AddPanelBottomSheetState();
}

class _AddPanelBottomSheetState extends State<AddPanelBottomSheet> {
  final _formKey = GlobalKey<FormState>();

  final _noPanelController = TextEditingController();
  final _noWbsController = TextEditingController();
  final _projectController = TextEditingController();
  final _noPpController = TextEditingController();
  final _progressController = TextEditingController();

  DateTime _selectedDate = DateTime.now();
  String? _selectedK3VendorId;
  bool _isLoading = false;
  bool _isSuccess = false;

  bool get _isAdmin => widget.currentCompany.role == AppRole.admin;

  @override
  void initState() {
    super.initState();
    if (!_isAdmin) {
      _selectedK3VendorId = widget.currentCompany.id;
    }
    _progressController.text = "0";
  }

  @override
  void dispose() {
    _noPanelController.dispose();
    _noWbsController.dispose();
    _projectController.dispose();
    _noPpController.dispose();
    _progressController.dispose();
    super.dispose();
  }

  Future<void> _pickDateTime() async {
    final date = await showDatePicker(
      context: context,
      initialDate: _selectedDate,
      firstDate: DateTime(2000),
      lastDate: DateTime(2101),
      initialEntryMode: DatePickerEntryMode.calendarOnly,
      builder: (context, child) {
        return Theme(
          data: ThemeData.light().copyWith(
            colorScheme: const ColorScheme.light(
              primary: AppColors.schneiderGreen,
              onPrimary: Colors.white,
              onSurface: AppColors.black,
            ),
            textButtonTheme: TextButtonThemeData(
              style: ButtonStyle(
                foregroundColor: WidgetStateProperty.resolveWith((states) {
                  if (states.contains(WidgetState.pressed)) {
                    return Colors.white;
                  }
                  return AppColors.schneiderGreen;
                }),
                backgroundColor: WidgetStateProperty.resolveWith((states) {
                  if (states.contains(WidgetState.pressed)) {
                    return AppColors.schneiderGreen;
                  }
                  return Colors.transparent;
                }),
                side: WidgetStateProperty.resolveWith((states) {
                  if (states.contains(WidgetState.pressed)) {
                    return BorderSide.none;
                  }
                  return const BorderSide(color: AppColors.schneiderGreen);
                }),
                padding: WidgetStateProperty.all(
                  const EdgeInsets.symmetric(vertical: 12, horizontal: 24),
                ),
                shape: WidgetStateProperty.all(
                  RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(6),
                  ),
                ),
                textStyle: WidgetStateProperty.all(
                  const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    fontFamily: 'Lexend',
                  ),
                ),
              ),
            ),
            datePickerTheme: DatePickerThemeData(
              backgroundColor: Colors.white,
              headerBackgroundColor: AppColors.white,
              headerForegroundColor: AppColors.black,
              dividerColor: AppColors.grayLight,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
          ),
          child: child!,
        );
      },
    );
    if (date == null) return;

    final time = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(_selectedDate),
      builder: (context, child) {
        return Theme(
          data: ThemeData.light().copyWith(
            colorScheme: const ColorScheme.light(
              primary: AppColors.schneiderGreen,
              onPrimary: Colors.white,
              onSurface: Colors.black,
              surface: Colors.white,
            ),
            textButtonTheme: TextButtonThemeData(
              style: ButtonStyle(
                foregroundColor: WidgetStateProperty.resolveWith((states) {
                  if (states.contains(WidgetState.pressed)) {
                    return Colors.white;
                  }
                  return AppColors.schneiderGreen;
                }),
                backgroundColor: WidgetStateProperty.resolveWith((states) {
                  if (states.contains(WidgetState.pressed)) {
                    return AppColors.schneiderGreen;
                  }
                  return Colors.transparent;
                }),
                side: WidgetStateProperty.resolveWith((states) {
                  if (states.contains(WidgetState.pressed)) {
                    return BorderSide.none;
                  }
                  return const BorderSide(color: AppColors.schneiderGreen);
                }),
                padding: WidgetStateProperty.all(
                  const EdgeInsets.symmetric(vertical: 12, horizontal: 24),
                ),
                shape: WidgetStateProperty.all(
                  RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(6),
                  ),
                ),
                textStyle: WidgetStateProperty.all(
                  const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    fontFamily: 'Lexend',
                  ),
                ),
              ),
            ),
          ),
          child: child!,
        );
      },
    );
    if (time == null) return;

    setState(() {
      _selectedDate = DateTime(
        date.year,
        date.month,
        date.day,
        time.hour,
        time.minute,
      );
    });
  }

  Future<void> _savePanel() async {
    if (_isLoading || _isSuccess) return;
    if (!_formKey.currentState!.validate()) return;

    if (_isAdmin && _selectedK3VendorId == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Silakan pilih Vendor Panel (K3)'),
          backgroundColor: Colors.red,
        ),
      );
      return;
    }

    setState(() => _isLoading = true);

    final noPp = _noPpController.text.trim();

    // Cek duplikasi No. PP
    final isPpTaken = await DatabaseHelper.instance.isNoPpTaken(noPp);
    if (isPpTaken) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('No. PP sudah ada. Gunakan nomor lain.'),
            backgroundColor: Colors.red,
          ),
        );
        setState(() => _isLoading = false);
      }
      return;
    }

    try {
      final newPanel = Panel(
        noPp: noPp,
        noPanel: _noPanelController.text.trim(),
        noWbs: _noWbsController.text.trim(),
        project: _projectController.text.trim(),
        percentProgress:
            double.tryParse(_progressController.text.trim()) ?? 0.0,
        startDate: _selectedDate,
        createdBy: widget.currentCompany.id,
        vendorId: _selectedK3VendorId,
        // Default status sesuai permintaan
        statusBusbarPcc: "On Progress",
        statusBusbarMcc: "On Progress",
        statusComponent: "Open",
        statusPalet: "Open",
        statusCorepart: "Open",
      );

      await DatabaseHelper.instance.insertPanel(newPanel);

      if (_selectedK3VendorId != null) {
        await DatabaseHelper.instance.upsertPalet(
          Palet(panelNoPp: newPanel.noPp, vendor: _selectedK3VendorId!),
        );
        await DatabaseHelper.instance.upsertCorepart(
          Corepart(panelNoPp: newPanel.noPp, vendor: _selectedK3VendorId!),
        );
      }

      await DatabaseHelper.instance.upsertComponent(
        Component(panelNoPp: newPanel.noPp, vendor: 'warehouse'),
      );

      setState(() {
        _isLoading = false;
        _isSuccess = true;
      });
      await Future.delayed(const Duration(milliseconds: 1500));

      if (mounted) {
        widget.onPanelAdded(newPanel);
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: SingleChildScrollView(
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
              const Text(
                "Tambah Panel",
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w500),
              ),
              const SizedBox(height: 16),
              _buildTextField(
                controller: _noPanelController,
                label: "No. Panel",
              ),
              const SizedBox(height: 16),
              _buildTextField(controller: _noWbsController, label: "No. WBS"),
              const SizedBox(height: 16),
              _buildTextField(controller: _projectController, label: "Project"),
              const SizedBox(height: 16),
              _buildTextField(controller: _noPpController, label: "No. PP"),
              const SizedBox(height: 16),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: _buildTextField(
                      controller: _progressController,
                      label: "Progress (%)",
                      isNumber: true,
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(flex: 2, child: _buildDateTimePicker()),
                ],
              ),
              const SizedBox(height: 16),
              if (_isAdmin)
                _buildAdminVendorPicker()
              else
                _buildK3VendorDisplay(),
              const SizedBox(height: 32),
              _buildActionButtons(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildTextField({
    required TextEditingController controller,
    required String label,
    bool isNumber = false,
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
              if (value == null || value.isEmpty) {
                return '0-100';
              }
              final progress = int.tryParse(value);
              if (progress == null) {
                return '0-100';
              }
              if (progress < 0 || progress > 100) {
                return '0-100';
              }
            }
            // Validator untuk field non-angka
            if (!isNumber && (value == null || value.isEmpty)) {
              return 'Field ini tidak boleh kosong';
            }
            return null; // Lolos validasi
          },

          decoration: InputDecoration(
            hintText: 'Masukkan $label', // Tambahkan placeholder
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

  Widget _buildDateTimePicker() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Waktu Mulai Pengerjaan",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        InkWell(
          onTap: _pickDateTime,
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

  Widget _buildAdminVendorPicker() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Vendor Panel (K3)",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 12),
        Wrap(
          children: widget.k3Vendors.map((vendor) {
            return _buildOptionButton(
              label: vendor.name,
              selected: _selectedK3VendorId == vendor.id,
              onTap: () => setState(() => _selectedK3VendorId = vendor.id),
            );
          }).toList(),
        ),
      ],
    );
  }

  Widget _buildK3VendorDisplay() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          "Vendor Panel (K3)",
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w400),
        ),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: AppColors.grayLight,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: AppColors.grayLight),
          ),
          child: Text(
            widget.currentCompany.name,
            style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w300),
          ),
        ),
      ],
    );
  }

  Widget _buildOptionButton({
    required String label,
    required bool selected,
    required VoidCallback onTap,
  }) {
    final Color borderColor = selected
        ? AppColors.schneiderGreen
        : AppColors.grayLight;
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        margin: const EdgeInsets.only(right: 8, bottom: 12),
        decoration: BoxDecoration(
          color: selected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : Colors.white,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          label,
          style: const TextStyle(fontWeight: FontWeight.w300, fontSize: 12),
        ),
      ),
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
              style: TextStyle(color: AppColors.schneiderGreen, fontSize: 12),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: ElevatedButton(
            onPressed: _savePanel,
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
