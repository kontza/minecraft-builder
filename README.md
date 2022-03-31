# Introduction
A companion app for my `minecraft-ansible` repository.

# How to Use
1. Run
    ```
    $ go install
    ```
1. `cd` into `minecraft-ansible`.
1. Run
    ```
    $ minecraft-builder group_vars/all
    ```

# What Does It Do?
It reads the contents of the given YAML and provides easy means to update it. The screen is divided into three parts: _Services_, _Settings_, and _Info_. _Services_ contains a list of detected Minecraft settings from the input YAML. _Settings_ contains a form in which you can edit the fields of the YAML. _Info_ tries to given some help on the currently active control's function.

# TODO
- _Server JAR_ -field should be a dropdown containing all JARs found in the current directory.
- _Server port_ -field's validator should check other settings' ports so that there won't be a clash.